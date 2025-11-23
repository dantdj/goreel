# goreel
A YouTube-style application that I'm working on to explore some queueing
mechanisms, as well as more exploring of Postgres in a web application.

## Callouts
* Stores raw and processed video data in blob storage
* Processes video data using FFMPEG
* Uses RabbitMQ for message queuing for video processing
  * This is currently more to mess around with queues than anything else,
    though it is handy to have a system for throttling the amount of video
    processing going on at once by limiting the number of messages processed
    at once.

## To do

* Database client to deal with keeping track of locations of videos in blob storage, and linking them to an ID
* Basic account system, using OAuth2 for logins
* Ability to favourite videos
* Video details - view count, number of likes, etc