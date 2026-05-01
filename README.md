### 1. The Framework: Gin

*   **The Justification:** Go's standard library is powerful, however Gin provides a highly performant, lightweight routing engine that greatly reduces boilerplate code. Coming from heavier, multi-layered architectures with strict MVC constraints (Java), Gin is refreshing because it stays out of my way. It allows me to strictly enforce my own "Clean Architecture" (separating handlers, services, and repositories) without forcing a specific directory structure on you.
*   **The Trade-off:** The drawback of Gin is that it is not "batteries-included." I will have to manually integrate third-party libraries for things like ORMs, structured logging, or complex validation.
*   **Why it fits the project:** For a 3-stage iterative project, I need a framework that lets I start small and scale up. Gin’s middleware support makes it incredibly easy to snap in the required "Authentication & Authorization" layer seamlessly across all routes.

### 2. The Database: MySQL
My instinct to use MySQL is perfectly aligned with the project requirements. The key here is to lean into the data domain.

*   **The Justification:** The system manages "Users, Teams, and Digital Assets" with "granular access control." This data is inherently highly relational. Users belong to Teams. Folders belong to Users. Notes belong to Folders. A relational database with strict ACID compliance guarantees that these relationships remain intact. If I used a NoSQL database (like MongoDB), I would risk data anomalies when, for example, a Manager is deleted but their sub-resources are left orphaned. 
*   **The Trade-off:** Relational databases like MySQL are generally harder to scale horizontally across multiple servers compared to NoSQL databases. If the system were to suddenly ingest millions of unstructured "Notes" per second, a document store might handle the write-load better.
*   **Why it fits the project:** The immediate priority is structural integrity and complex querying (e.g., "Find all assets in Folder X that belong to Team Y where User Z has Manager access"). SQL is purpose-built for exactly this type of relational algebra.

### 3. The API Protocol: REST
REST is a highly defensible choice, especially for the initial stages of a microservices build. 

*   **The Justification:** REST provides a clean, universally understood contract between the client and server. It relies on standard HTTP methods (GET, POST, PUT, DELETE) and utilizes standard HTTP status codes, which directly fulfills the project's "Error Handling" requirement for graceful degradation. It is also highly cacheable at the network level. 
*   **The Trade-off:** REST can suffer from over-fetching or under-fetching data. When you're used to seeing complex GraphQL queries where the client asks for exactly the fields it needs, REST can feel a bit rigid, sometimes requiring multiple round-trips to the server to stitch together a complete view of a "Team" and all its "Assets."
*   **Why it fits the project:** Setting up REST is significantly faster than defining complex GraphQL schemas or writing gRPC `.proto` files. It allows I to deliver Stage 1 quickly and establish a baseline. I can always introduce gRPC later specifically for internal service-to-service communication if the system scales out.