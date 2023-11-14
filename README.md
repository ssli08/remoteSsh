# remoteSsh
Used to access server through Jumper Host
```
./remoteSsh -h
Example Usage:
	rssh -p proj -r remoteHost 
	rssh -p proj -R role -c cmd

Usage:
  rssh [flags]
  rssh [command]

Available Commands:
  export      export db record to csv format file
  help        Help about any command
  import      import/update instances list in DB from api or file
  initdb      initial db used to store instance info, currently support sqlite3 and mysql
  net         network latency test
  pwd         port forword

Flags:
  -c, --cmd              switch for run cmd in batch
  -d, --dest string      copy file to dest path
  -f, --fcopy            copy files to remote host
  -h, --help             help for rssh
  -s, --print            show instances list
  -p, --project string   project server to connect and show server list, options: gwn|gdms|ipvt
  -r, --rh string        remote host to be connected
  -R, --role string      required if you get commands executed in batch, options: web|ssh|turn
      --rport string     remote host ssh port (default "26222")
      --ru string        remote host ssh user (default "ec2-user")

Use "rssh [command] --help" for more information about a command.
```
