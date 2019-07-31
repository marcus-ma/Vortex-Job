# Vortex-Job
Vortex-Job is a scheduled job tool, based on MonogoDB
## dependency
MonogoDB and its package [go.mongodb.org/mongo-driver/mongo]
<br>
other package [github.com/gorhill/cronexpr]
## Used way
### Job create
curl -H "Content-Type:application/json" http://127.0.0.1:9999/job/add -X POST -d '{"name":"job1","command":"echo i am job1","cron_expr":"*/5 * * * * * *"}'
### Job stop
curl -H "Content-Type:application/json" http://127.0.0.1:9999/job/stop -X POST -d '{"name":"job1"}'
### Job reboot
curl -H "Content-Type:application/json" http://127.0.0.1:9999/job/reboot -X POST -d '{"name":"job1"}'
### Job listAll
curl http://127.0.0.1:9999/job/list 
### Job modify
curl -H "Content-Type:application/json" http://127.0.0.1:9999/job/modify -X POST -d '{"id":"hgu8979834hgtr09","name":"job2","command":"echo i am job2","cron_expr":"*/5 * * * * * *"}'
