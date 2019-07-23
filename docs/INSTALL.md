# Installation

The installation process of the Coconut servers takes multiple steps.

0. Ensure you have correctly installed and configured docker and docker-compose.

1. Firstly get the copy of the repo with either `git clone git@github.com:nymtech/nym.git` or `go get github.com/nymtech/nym`. Using the first command is recommended in case there were any issue with go tools.

2. Build the entire system by invoking `make localnet-build`.

3. If you wish to modify keys used by issuers or their configuration, modify files inside `localnetdata/` directory. Currently those files are being coppied into docker volumes.

4. Run the system with `make localnet-start`

## Client GUI:

There exists also a GUI version of client used to demonstrate the system. You can either use a binary provided on the release page or build it from source. In order to do this, firstly you have to install the Qt bindings by following the instructions here:

[https://github.com/therecipe/qt/wiki/Installation-on-Linux](https://github.com/therecipe/qt/wiki/Installation-on-Linux)

Then navigate to `client/gui` directory and invoke one of the following command: `qtdeploy build desktop` Note that this command was only tested on the Linux systems.
