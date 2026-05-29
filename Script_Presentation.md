This is my take on the Team Management app.


During development, I realized that the number of active user is only 1, and that's me. So my solution was Redis pub/sub to handle audit logging, while processing CSV bulk user import is only 5 Goroutine, using a high end Message Brokers such as RabbitMQ or Kafka here would be like Taylor Swift moving from town to town in a private jet, completely unnecessary.

So, in the future when my app hopefully hits a few million users, I will implement Kafka to replace what Redis queue has been doing in my app. This will increase the complexity of the code, as well as the demand on server hardware, but that is the price to pay to increase durability to accommodate more users. (All these under the slide with picture of Redis pointing to Kafka logo)


Another upgrade path for my project would be to improve the user's ability to write notes, I can have it accept media files. But with that, I will need to offload the media to S3 Bucket, the Go API will generate a presigned URL, allowing the user to directly send their assets to S3, while my database will only hold the metadata.

Of course, I can expand further on this by also accepting complex text files such as tables, Kanban board, embedded links, or even drawings for example. But with that I will need to migrate the note function to a NoSQL such as MongoDB, Mongo here will keep the note as however the user make it to be.

But, a question arises, why don't I just migrate everything to Mongo? That's because I need the ACID nature of a SQL database to keep the system in check. Since you all have done the same project as me, you would know that we need strict relationship between entities such as Team, Members, Managers, Notes and Folders. A NoSQL database would struggle to handle this complexity, what if we need to get all the notes that 1 user was shared? In SQL? 1 single SELECT line. In Mongo? We are forced to stitch the data together in Go, introducing unnecessary complexity and latency. But we can save everything as a JSON right? Ok then what if someone changes their username? In SQL that's 1 line, in NoSQL you will need to change the name in every single document that you save the user, the server break down? Half the note will have the user's old name, half is new name.


That's my take on the project, thank you for listening

### Link to canva: https://www.canva.com/design/DAHK6n9Yra4/hFo1R7KflhWBNVK-41JO6Q/edit