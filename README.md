# ixos-update

A CLI tool to update IxOS on an Ixia Native IxOS chassis.

Built and tested on Go version 1.5

See [golang](https://golang.org/) for more information on this programing language

The included binary has been built and tested for windows 10

The materials here are not part of any Ixia product and, therefore, are not covered by Ixia Support.



## Usage
```
updateIxOS.exe --host 192.168.1.1 --version 9.10.2000.11


--host      <IP address of your chassis>
--user      <username> , defaults to admin 
--password  <username> , defaults to admin 
--package   <update package name> , needs to be in the same directory as the update program
--version   <targeted IxOS version> , if your chassis is able to online update
```

## Compile on Windows or Linux:
* install go
* fork or download github.com/lvpcguru/ixos-update
* install dependencies 
* 
```
curl -L -O https://github.com/BrennenWright/ixos-update/archive/master.zip
tar -xf master.zip 
cd ixos-update-master
go get github.com/pkg/sftp
go get github.com/schollz/progressbar"
go get golang.org/x/crypto/ssh"
go build update.go
```



## To cross compile go applications

* set your GOARCH environment variable to the target 386(32-bit x86), amd64 (64-bit x86)
* set your GOOS environment variable to to the target linux,windows,android,freebsd or darwin(for macs)
* and then go build update.go as usual
* see here for more info: https://golang.org/doc/install/source
