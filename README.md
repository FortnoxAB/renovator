# renovator

```
$ renovator -h
NAME:
   renovator - Split renovate workload across multiple agents using redis

USAGE:
   renovator [global options] command [command options]

COMMANDS:
   master   run renovate master
   agent    run renovate agent
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --loglevel value  available levels are: panic,fatal,error,warning,info,debug,trace (default: "info")
   --log-json        log in json (default: false)
   --help, -h        show help
```

```
$ renovator master -h
NAME:
   renovator master - run renovate master

USAGE:
   renovator master [command options]

OPTIONS:
   --redis-url value      redis url, ex redis://[[username]:[password]]@localhost:6379/0 or rediss://[[username]:[password]]@localhost:6379/0 for tls/ssl connections
   --leaderelect          run leader election (default: false)
   --election-ttl value   leader election ttl (default: 10s)
   --schedule value       Run discovery on a schedule instead of onetime, value is a standard cron string
   --run-first-time       run discovery directly, only applicable if a schedule is provided (default: false)
   --port value           webserver port for pprof and metrics (default: "8080")
   --kafka-brokers value  listen to bitbucket webhook transported over kafka
   --help, -h             show help
```

```
$ renovator agent -h
NAME:
   renovator agent - run renovate agent

USAGE:
   renovator agent [command options]

OPTIONS:
   --redis-url value          redis url, ex redis://[[username]:[password]]@localhost:6379/0 or rediss://[[username]:[password]]@localhost:6379/0 for tls/ssl connections
   --max-process-count value  Defines the maximum amount of simultaneous renovate processes (default: 1)
   --port value               webserver port for pprof and metrics (default: "8080")
   --help, -h                 show help
```
