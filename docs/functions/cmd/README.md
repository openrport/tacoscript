# cmd.run

The task `cmd.run` executes an arbitrary command in a shell of a host system. It has following syntax:


    create-file:
      cmd.run:
        - name: echo 'data to backup' >> /tmp/data2001-01-01.txt
    backup-data:
      cmd.run:
        - names:
            - tar czf /dumps/data2001-01-01.txt.tar.gz /tmp/data2001-01-01.txt
            - md5sum /dumps/data2001-01-01.txt.tar.gz >> /dumps/data2001-01-01.txt.tar.gz.md5
        - cwd: /tmp
        - require:
            - create-file
        - creates:
            - /tmp/data2001-01-01.txt.tar.gz
        - env:
            - PASSWORD: bunny
        - shell: bash
        - unless: service backup-runner status
        - onlyif: date +%c|grep -q "^Thu"

You can interpret this script as following:

The desired state of the script 'create-file' is a file at /tmp/data2001-01-01.txt. To achieve it, the tacoscript binary should execute a task of type 'cmd.run'. This task type executes command `echo 'data to backup' >> /tmp/data2001-01-01.txt` in the shell of the host system.
The second desired state of the script 'backup-data' is creation of 2 files: `/dumps/data2001-01-01.txt.tar.gz` and it's md5 sum file at `/dumps/data2001-01-01.txt.tar.gz.md5`.

It should be achieved by executing a task of type 'cmd.run' which requires 2 shell commands:

- `tar czf /dumps/data2001-01-01.txt.tar.gz /tmp/data2001-01-01.txt`
- `md5sum /dumps/data2001-01-01.txt.tar.gz >> /dumps/data2001-01-01.txt.tar.gz.md5`

The commands will be executed in the following context:
- current working directory (cwd) will be `/tmp`
- env variables list will contain PASSWORD with value `bunny`
- as shell `bash` will be selected

Both commands will be only executed after `create-file` script. So the tacoscript interpreter will make sure that the `create-file` script is executed before `backup-data` and only if it was successful.

The commands will be executed only under the following conditions:
- The file `/tmp/data2001-01-01.txt.tar.gz` should be missing (to avoid data overriding)
- The service `service backup-runner` should not be running
- Today is Thursday

## Task parameters

### name

[string] type

    create-file:
      cmd.run:
        - name: echo 'data to backup' >> /tmp/data2001-01-01.txt
        

Name describes a single executable command. In the example above the tacoscript interpreter will run `echo 'data to backup' >> /tmp/data2001-01-01.txt` command. 

### names
[array] type

    backup-data:
      cmd.run:
        - names:
            - tar czf /dumps/data2001-01-01.txt.tar.gz /tmp/data2001-01-01.txt
            - md5sum /dumps/data2001-01-01.txt.tar.gz >> /dumps/data2001-01-01.txt.tar.gz.md5

Name contains the list of commands. All commands in the task will be executed in the order of appearance. If one fails, the whole execution will stop. All commands inside one task will be executed in the same context, which means in the current working directory, with same env variables, in the same shell and under same conditions.

If you want to change context of a command (e.g. use another shell), you should create another task e.g.


    backup-data:
      cmd.run:
        - name: mycmd in shell 'bash'
        - shell: bash
      cmd.run:
        - name: mycmd in shell 'sh'
        - shell: sh


The `names` parameter with a single value has the same meaning as `name` field. 

### cwd
[string] type

    backup-data:
      cmd.run:
        - cwd: /tmp

The `cwd` parameter gives the current working directory of a command. This value is quite useful if you want to use relative paths of files that you provide to your commands. 

For example imagine following file structure:

    C:\Some\Very\Long\Path
      someData1.txt
      someData2.txt


You want pack someData1.txt and someData2.txt with the zip.exe binary. You can do it with the script as:

    backup-data:
      cmd.run:
        - names:
            - tar.exe -a -c -z -f C:\Some\Very\Long\Path\someData1.txt.tar.gz C:\Some\Very\Long\Path\someData1.txt
            - tar.exe -a -c -z -f C:\Some\Very\Long\Path\someData2.txt.tar.gz C:\Some\Very\Long\Path\someData2.txt


Obviously we have quite long strings here. 
You can also change your working directory and use relative paths to get the same result:

    backup-data:
      cmd.run:
        - cwd: C:\Some\Very\Long\Path\
        - names:
            - tar.exe -a -c -z -f someData1.txt.tar.gz someData1.txt
            - tar.exe -a -c -z -f someData2.txt.tar.gz someData2.txt

### shell
[string] type

Shell is a program that takes commands from input and gives them to the operating system to perform. Known Linux shells are [bash](https://www.gnu.org/software/bash/), [sh](https://www.gnu.org/software/bash/), [zsh](https://ohmyz.sh/) etc. Windows supports [cmd.exe](https://ss64.com/nt/cmd.html) shell. 

If you don't specify this parameter, tacoscript will use the default golang [exec function](https://golang.org/pkg/os/exec/)
which intentionally does not invoke the system shell and does not expand any glob patterns or handle other expansions, pipelines, or redirections typically done by shells.
 
 To expand glob patterns, you can specify the `shell` parameter, in this case you should take care to escape any dangerous input.
 
 **Note that if you use `cmd.run` task type without the `shell` parameter, usual patterns like pipelines and redirections won't work.** 
 
 If you specify a `shell` parameter, tacoscript will run your task commands as a '-c' parameter under Unix and '/C' parameter under Windows:
 
 The script below:
 
     backup-data:
       cmd.run:
       - names:
         - touch serviceALock.txt
         - tar cvf somedata.txt.tar somedata.txt
         - rm serviceALock.txt
       - shell: bash
       
will run as:

    bash -c touch serviceALock.txt
    bash -c tar cvf somedata.txt.tar somedata.txt
    bash -c rm serviceALock.txt
    
The script below:
 
     backup-data:
       cmd.run:
       - name: date.exe /T > C:\tmp\my-date.txt
       - shell: cmd.exe
       
will run as:

    cmd.exe \C date.exe /T > C:\tmp\my-date.txt

### user
[string] type

    create-user-file:
      cmd.run:
        - user: www-data
        - touch: data.txt

The `user` parameter allows to run commands as a specific user. In Linux systems this will require sudo rights for the tacoscript binary. In Windows this command will be ignored.
  Switching users allows to create resources (file, services, folders etc) with the ownership of the specified user. 

After running the above script, tacoscript will create a `data.txt` file with the ownership of `www-data` user.

    sudo tacoscript tacoscript.yaml
    ls -la
    #output will be
    -rw-r--r--    4 root  root          128 Jul 15  2019 tacoscript.yaml
    -rw-r--r--    1 root  root          3550 Oct 30  2019 tacoscript
    -rw-r--r--@   1 www-data  www-data  0 Apr 23 09:22 data.txt
    
    
### env
[keyValue] type 

The `env` parameter is a list of key value parameters where key represents the name of an environment variable and value it's content. Env variables are parameters which are set from the outside of a running program and can be used as configuration data.

    save-date:
      cmd.run:
        - name: psql
        - env:
            - PGUSER: bunny
            - PGPASSWORD: bug

In this example the psql will read login and password from the corresponding env variables and connect to the database without any input parameters or configuration data.

### require
see [require](../../general/dependencies/require.md)

### creates
see [creates](../../general/conditionals/creates.md)

### onlyif
see [onlyif](../../general/conditionals/onlyif.md)

### unless
see [onlyif](../../general/conditionals/unless.md)
