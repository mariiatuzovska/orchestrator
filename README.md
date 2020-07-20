# orchestrator

*Remote/Local service/node management solution*

![](https://github.com/mariiatuzovska/orchestrator/blob/master/gophercomplex.jpg)

# orchestrator-manager

*API service for orchestrator's services management*

## Progress: 85%

- [x] Local access for *darwin*;
- [x] Local access for *linux*;
- [ ] Local access for *windows*;
- [x] Remote access *systemD*;
- [x] Remote access *launchD*;
- [ ] Remote access for *windows svc*;
- [x] Orchestrators management by configuration file;
- [x] Data output (Service, Node);
- [x] Data filtering (Service, Node);
- [x] Registering/editing/deleting service;
- [x] Registering/editing/deleting new node;
- [x] Services status output;
- [x] CMD app.

## cmd application

```
NAME:
   orchestrator-manager - Is a management service for discovering local/remote services activity

USAGE:
   main [global options] command [command options] [arguments...]

VERSION:
   0.0.6

AUTHOR:
   Tuzovska Mariia

COMMANDS:
   s        Start
   help, h  Shows a list of commands or help for one command

OPTIONS:
   -s value      Path to services configuration file (default: "./service-configuration.json")
   -n value      Path to nodes configuration file (default: "./node-configuration.json")
   --host value  Host (default: "127.0.0.1")
   --port value  Port (default: "6000")

GLOBAL OPTIONS:
   --help, -h     show help
   --version, -v  print the version

COPYRIGHT:
   2020, mariiatuzovska
```

# orchestrator-installer

*cmd application for remote installation of some service*

## Progress: 50%

- [x] Local access for *darwin*;
- [x] Local access for *linux*;
- [ ] Local access for *windows*;
- [x] Remote access *systemD*;
- [x] Remote access *launchD*;
- [ ] Remote access for *windows svc*;
- [ ] Installing service for *darwin*;
- [x] Installing service for *linux* (*.deb*);
- [ ] Installing service for *windows*.

## cmd application

```
NAME:
   orchestrator-installer - Is a cmd application for remote service installation

USAGE:
   main [global options] command [command options] [arguments...]

VERSION:
   0.0.6

AUTHOR:
   Tuzovska Mariia

COMMANDS:
   i        Install service
   help, h  Shows a list of commands or help for one command

OPTIONS:
   --package value, -p value           Path to pacakage (default: "./")
   --os value, -o value                OS (default: "linux")
   --host value                        Host
   --port value                        Port (default: "22")
   --user value, -u value              User (default: "root")
   --key value, --ssh value, -k value  SSH key path (default: "~/.ssh/id_rsa")
   -f value                            PassPhrase

GLOBAL OPTIONS:
   --help, -h     show help
   --version, -v  print the version

COPYRIGHT:
   2020, mariiatuzovska
```