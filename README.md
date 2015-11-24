go KX3 panadapter
=================

alpha (still development)

<img src="https://dl.dropboxusercontent.com/u/673746/Screenshots/2015-10-01%2009.31.59.png"/>

<a href="https://youtu.be/scCF7HCArNY">Movie</a>

Features
========

Main features:

 * FFT and waterfall view (bandwidth depends on your soundcard)
 * Change center (local oscillator) frequency by clicking FFT view

And some features included for utility:

 * WebSocket server which can manipulate KX3's text buffer for sending CW/RTTY
 * HTTP server which serves some utility (eg. Sending CW code from Mac)


Requirements
============

 * *Stereo* LINE-IN Sound Card
 * KX3 USB Cable connected to ACC1

Configure
=========

See [config.json]( ./config.json )

 1. Change "port" -> "name" to path of USB serial device file (typically /dev/tty.usbserial-*)
 2. Check the "port" -> "baudrate" is same as KX3's RS-232 setting
 3. Change "input" -> "name","samplerate" to match as your input device (or just remove to use system default)

Usage
=====

Run:

	./go-KX3-panadapter --config ./config.json

go-KX3-panadapter works as HTTP server. So open shown http: URL by WebGL supported browsers (Google Chrome / Mozilla Firefox).


Development
===========

Most code is written in golang. I don't build binary because this project is still alpha.

## Mac OS X

Install prerequisites:
```
brew install portaudio
```

Download sources:
```
go get
```

Build and run:
```
go build && ./go-KX3-panadapter
```
