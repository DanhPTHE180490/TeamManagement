### A. Techinal justification
### 1. The Framework: Gin

*   **The Justification:** Go's standard library is powerful, however Gin provides a highly performant, lightweight routing engine that greatly reduces boilerplate code. Coming from heavier, multi-layered architectures with strict MVC constraints (Java), Gin is refreshing because it stays out of my way. Approaching backend development with a rigorous automation testing mindset where testability is paramount, Gin allows for the strict enforcement of "Clean Architecture" (separating handlers, services, and repositories). This makes unit and integration testing a breeze without forcing a specific directory structure.
*   **The Trade-off:** The drawback of Gin is that it is not "batteries-included." I will have to manually integrate third-party libraries for things like ORMs, structured logging, or complex validation.
*   **Why it fits the project:** For a 3-stage iterative project, I need a framework that lets me start small and scale up. Gin’s middleware support makes it incredibly easy to snap in the required "Authentication & Authorization" layer seamlessly across all routes.

### 2. The Database: MySQL

*   **The Justification:** The system manages "Users, Teams, and Digital Assets" with "granular access control." This data is inherently highly relational. Users belong to Teams, Folders belong to Users, Notes belong to Folders. A relational database with strict ACID compliance guarantees that these relationships remain intact. If I used a NoSQL database (like MongoDB), I would risk data anomalies when, for example, a Manager is deleted but their sub-resources are left orphaned. 
*   **The Trade-off:** Relational databases like MySQL are generally harder to scale horizontally across multiple servers compared to NoSQL databases. If the system were to suddenly ingest millions of unstructured "Notes" per second, a document store might handle the write-load better, or maybe there is a network error between the client and the server, both of which don't really apply to a localhost project.
*   **Why it fits the project:** The immediate priority is structural integrity and complex querying (e.g., "Find all assets in Folder X that belong to Team Y where User Z has Manager access"). Furthermore, the inherent read-scaling limitations of SQL were completely bypassed in this project by implementing a distributed Redis caching layer in front of the database.

### 3. The API Protocol: REST

*   **The Justification:** REST provides a clean, universally understood contract between the client and server. It relies on standard HTTP methods and utilizes standard HTTP status codes, which directly fulfills the project's "Error Handling" requirement for graceful degradation. It is also highly cacheable at the network level. 
*   **The Trade-off:** REST can suffer from over-fetching or under-fetching data. When you're used to seeing complex GraphQL queries where the client asks for exactly the fields it needs, REST can feel a bit rigid, sometimes requiring multiple round-trips to the server to stitch together a complete view of a "Team" and all its "Assets."
*   **Why it fits the project:** Setting up REST is significantly faster than defining complex GraphQL schemas or writing gRPC `.proto` files. It allows me to deliver Stage 1 quickly and establish a baseline. I can always introduce gRPC later specifically for internal service-to-service communication if the system scales out.

### 4. Cache & Event Broker: Redis

*   **The Justification:** Redis serves a dual purpose in this architecture. First, it acts as a high-speed caching layer for frequently accessed data, drastically reducing database IOPS. Second, it acts as a Message Broker utilizing Pub/Sub channels to decouple the Audit Logging system from the main HTTP request thread.

*   **The Trade-off:** Introducing a distributed cache adds infrastructure complexity and introduces the classic "cache invalidation" problem, requiring strict discipline in the repository layer to clear cache keys during updates to prevent stale data.

*   **Why it fits the project:** It transforms a standard CRUD API into a highly performant, production-ready system. Offloading the audit logs to a background worker ensures that end-users experience zero latency degradation, even when the system is heavily tracking security and asset events.

### 5. Infrastructure: Docker

*   **The Justification:** Containerizing the Go API, MySQL database, and Redis cache ensures absolute environment parity. The classic "it works on my machine" issue is completely eliminated.

*   **The Trade-off:** Introduces a learning curve and slight resource overhead for local development compared to running bare-metal Go binaries.

*   **Why it fits the project:** It allows the entire microservice ecosystem to be spun up, networked, and tested with a single docker compose up command. This makes deployment trivial and provides a seamless review experience for other developers.

### B. How to run

### 1. Clone the Repository

`git clone [https://github.com/danhpthe180490/teammanagement.git](https://github.com/danhpthe180490/teammanagement.git)`
`cd teammanagement`

### 2. Evironment Configuration

`cp .env.example .env`

(Ensure you set a secure JWT_SECRET and confirm the DB_DSN and REDIS_URL match your Docker setup).

### 3. Run with Docker

`docker compose up -d --build`

The app will be available at http://localhost:8080.

### C. API Documentation

Once the Docker containers are running, navigate to:
http://localhost:8080/swagger/index.html

### D. Testing

This project includes both isolated unit tests and full-suite integration tests (utilizing miniredis and test database transactions).

Run all tests:

`go test ./... -v`

## E. Architecture Decisions & Future Scaling

While this application currently serves a small user base, the backend is designed with a decoupled, event-driven foundation to allow for seamless enterprise scaling. Below are the design choices for V1, and the targeted upgrade paths for a high-traffic environment.

### 1. The Current State: Event-Driven Background Jobs
**Decision:** Implemented Redis Pub/Sub instead of a heavy message broker.
**Reasoning:** To handle asynchronous tasks (like Audit Logging), a queue is necessary so the HTTP response isn't delayed. However, deploying a heavy-duty broker like Kafka or RabbitMQ for a V1 product with a small user base is premature optimization. Redis Pub/Sub provides a lightweight, in-memory "fire-and-forget" orchestrator that handles background Goroutine workers perfectly without unnecessary hardware overhead.

**The Upgrade Path:** As user count scales into the millions, Redis Pub/Sub will be replaced by **Kafka**. This introduces a disk-backed Dead Letter Queue, ensuring absolute data durability (zero lost audit logs) even during server outages.

### 2. Media Scaling: AWS S3 & Presigned URLs
**Decision:** Offloading media storage from the Go API.
**Reasoning:** As Note-taking capabilities expand to accept heavy media (images and 4K video), routing large files through the Go server will exhaust memory and crash the API. 
**The Upgrade Path:** The Go API will generate **Presigned URLs**. The frontend will use these URLs to upload heavy assets directly to an AWS S3 Bucket, bypassing our servers entirely and saving massive bandwidth, while MySQL simply stores the URL string as a reference.

### 3. Polyglot Persistence: MySQL + MongoDB
**Decision:** Maintaining strict ACID compliance with MySQL for core relationships, rather than defaulting entirely to NoSQL.
**Reasoning:** The core of this system relies on strict relational logic (Users, Teams, Roles, and Folder Permissions). Moving this to a NoSQL database would force the Go application to manually stitch together complex relationships (simulating `JOIN`s), introducing massive latency and data synchronization risks (e.g., updating a username across thousands of decentralized documents).

**The Upgrade Path:** We will maintain MySQL for core entity relationships and access control. However, to support complex, highly-nested Note data (Kanban boards, rich-text embeds, unstructured metadata), the Note content itself will be offloaded to **MongoDB**. This hybrid approach uses the exact right tool for the job: SQL for relationships, NoSQL for scale and flexibility.