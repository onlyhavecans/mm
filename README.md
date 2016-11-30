# mm

mm stands for Mu*Me

mm is the muck/mush/mud/Mu* equivalent of [ii](http://tools.suckless.org/ii/), the FIFO and filesystem based irc client.

This is a rewrite of the [python version](https://github.com/onlyhavecans/mmPython) in go.

## Install

### Easy
* Install golang/go
* `go install github.com/onlyhavecans/mm/...`

### Easier
* Download binary for your OS from [Releases](https://github.com/onlyhavecans/mm/releases)
* (Optional) rename from `mm_<os>` to `mm`
* Place somewhere in your path

## Usage

`mm [--debug] [--nolog] [--ssl] [--insecure] <name> <server> <port>`

When run it creates the ~/muck directory and a subdirectory based on the `name`

In this directory it creates an `out` file which has the output, and a FIFO `in`

write to in, get muck from out.

If your Mu* supports ssl you can use the `--ssl` flag. Disable strick checking with `--insecure`

When you disconnect the program quits and rotates out to a timestamped log. You can disable this with `--nolog`

## Advanced usage tips
- I wrote an vim plugin called [mm.vim](https://github.com/onlyhavecans/mm.vim) that uses vim as the inut system. Check it out for more vim tips and usage.
- Use screen or tmux to split your screen and then multitail to read out with all the coloring you need.
- simple shell or perl scripts can be used to read from out and automate tasks
- A simple shell script wrapper could set up your env and auto log you in
- in fact, most all tips and tricks for ii will work for mm including now to handle multiple sessions with one screen, ect
