# cblogger

## Installation

### 0. Fork the repo

You *will* need to modify the source `cblogger.go`. This repo is only a scaffold.
To be able to versionize and push your changes you should fork the repo and put it under your prefered version control.

### 1. Get the source to your server

The easiest would be to just clone the repo to your server.

```git clone <repo url>```

Of course you can also integrate it with your prefered deployment tools.

### 2. The init script

The repo contains an example init script that works on AWS AMI images.
If you need to change the options with which the logger starts, you can edit it here.

The supported parameters are:

`--logdir`
Where to put the csv files.
The default is `./`.

`--logfile`
The name of the current log file.
The default is `csv`.

`--time`
The suffix each file gets. The option accepts the golang time [strings](https://golang.org/pkg/time/#Time.Format).
The default is `2006-01-02_15`.

`--location`
The time location.
The default is `UTC`.

`--s3`
The s3 bucket name. By default this is unset, which means no rotation to s3.
If you don't setup s3 rotation old files are not deleted.
If you use s3 rotation only the last 12 files are kept locally.
If rotation to s3 fails, the files gets renamed to `failed_<filename>` and gets excluded from deletion.

`--interval`
How often to move the files out. The default is 3600 seconds.
If you change the rotation interval you will have to change the `--time` parameter as well to avoid overwriting your files.

`--port`
The port on which the cblogger runs.
The default is 3000.

### 3. Installation

If you use an AWS AMI image the installation should be as simple as

```sudo bash bootstrap.sh```

This will do the following steps:

- update your system
- install the golang package
- install and configure nginx, add nginx to default runlevel and start nginx
- install the deamonize software
- build the binary, configure the init script, add to default runlevel and the start the  software

If you want to use the hourly push to s3 you also need to configure the aws cli tool to use your credentials.

```bash
sudo su -
aws configure
<follow the dialog>
```

Please be aware that this setup will run this software as root. If you are not comfortable with it, you need to edit the init script and permissions.

Also if you want to use another webserver than nginx, you will need to perform the steps described in ```bootstrap.sh``` and replace the nginx steps with appropriate configuration steps for your prefered webserver.


## Customizing

If you want to add other placeholders to your csv files, you can do that by editing the source code. There is an array called `paramList` which holds all params you are interested in.

A little example: We set a callback in our adjust dashboard that reads:

```
http://your-domain/?app_name=my_app&event_name={event}
```

adjust would replace {event} with an appropriate value, resulting in a callback that looks something like:

```
http://your-domain/?app_name=my_app&event_name=f0ob4r
```

Now say you want your CSV to look like this:

```
"event_name","app_name"
```

So your paramList needs to look like that:

```go
paramList = []string{
    "app_name", "event_name",
}
// trailing comma is important!
```

After you edited the source code, you can rebuild the binary and restart the software with

```
sudo bash build.sh
```
