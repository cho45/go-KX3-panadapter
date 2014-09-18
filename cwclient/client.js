
var App = angular.module('App', ['ui.bootstrap']);

App.factory('MorseDevice', function ($q, config) {
	var MORSE_CODES = [
		0, // 0 NUL
		parseInt("111010111010111", 2), // 1 SOH => CT / KA
		0, // 2 STH
		0, // 3 ETX
		parseInt("101010111010111", 2), // 4 EOT => SK
		0, // 5 ENQ
		parseInt("10101011101", 2), // 6 ACK => SN
		0, // 7 BEL
		0, // 8 BS
		0, // 9 HT
		parseInt("10111010111", 2), // 10 LF => AA
		0, // 11 VT
		0, // 12 FF
		0, // 13 CR
		0, // 14 SO
		parseInt("1110101011101110111", 2), // 15 SI => DO (WABUN)
		0, // 16 DLE
		0, // 17 DC1
		0, // 18 DC2
		0, // 19 DC3
		0, // 20 DC4
		0, // 21 NAK
		0, // 22 SYN
		0, // 23 ETB
		0, // 24 CAN
		parseInt("1110101010111010111", 2), // 25 EM => BK
		parseInt("111010111011101", 2), // 26 SUB => KN
		parseInt("111010111010101110101", 2), // 27 ESC => CL
		0, // 28 FS
		0, // 29 GS
		0, // 30 RS
		0, // 31 US
		0, // 32 " "
		parseInt("1110101110101110111", 2), // 33 "!"
		parseInt("101110101011101", 2), // 34 """
		0, // 35 "#"
		parseInt("10101011101010111", 2), // 36 "$"
		0, // 37 "%"
		parseInt("10111010101", 2), // 38 "&" AS
		parseInt("1011101110111011101", 2), // 39 "'"
		parseInt("111010111011101", 2), // 40 "("
		parseInt("1110101110111010111", 2), // 41 ")"
		0, // 42 "*"
		parseInt("1011101011101", 2), // 43 "+"
		parseInt("1110111010101110111", 2), // 44 ","
		parseInt("111010101010111", 2), // 45 "-"
		parseInt("10111010111010111", 2), // 46 "."
		parseInt("1110101011101", 2), // 47 "/"
		parseInt("1110111011101110111", 2), // 48 "0"
		parseInt("10111011101110111", 2), // 49 "1"
		parseInt("101011101110111", 2), // 50 "2"
		parseInt("1010101110111", 2), // 51 "3"
		parseInt("10101010111", 2), // 52 "4"
		parseInt("101010101", 2), // 53 "5"
		parseInt("11101010101", 2), // 54 "6"
		parseInt("1110111010101", 2), // 55 "7"
		parseInt("111011101110101", 2), // 56 "8"
		parseInt("11101110111011101", 2), // 57 "9"
		parseInt("11101110111010101", 2), // 58 ":"
		parseInt("11101011101011101", 2), // 59 ";"
		0, // 60 "<"
		parseInt("1110101010111", 2), // 61 "=" BT
		0, // 62 ">"
		parseInt("101011101110101", 2), // 63 "?"
		parseInt("10111011101011101", 2), // 64 "@"
		parseInt("10111", 2), // 65 "A"
		parseInt("111010101", 2), // 66 "B"
		parseInt("11101011101", 2), // 67 "C"
		parseInt("1110101", 2), // 68 "D"
		parseInt("1", 2), // 69 "E"
		parseInt("101011101", 2), // 70 "F"
		parseInt("111011101", 2), // 71 "G"
		parseInt("1010101", 2), // 72 "H"
		parseInt("101", 2), // 73 "I"
		parseInt("1011101110111", 2), // 74 "J"
		parseInt("111010111", 2), // 75 "K"
		parseInt("101110101", 2), // 76 "L"
		parseInt("1110111", 2), // 77 "M"
		parseInt("11101", 2), // 78 "N"
		parseInt("11101110111", 2), // 79 "O"
		parseInt("10111011101", 2), // 80 "P"
		parseInt("1110111010111", 2), // 81 "Q"
		parseInt("1011101", 2), // 82 "R"
		parseInt("10101", 2), // 83 "S"
		parseInt("111", 2), // 84 "T"
		parseInt("1010111", 2), // 85 "U"
		parseInt("101010111", 2), // 86 "V"
		parseInt("101110111", 2), // 87 "W"
		parseInt("11101010111", 2), // 88 "X"
		parseInt("1110101110111", 2), // 89 "Y"
		parseInt("11101110101", 2), // 90 "Z"
		0, // 91 "["
		0, // 92 "\"
		0, // 93 "]"
		0, // 94 "^"
		parseInt("10101110111010111", 2), // 95 "_"
		0, // 96 "`"
		0, // 97 "a"
		0, // 98 "b"
		0, // 99 "c"
		0, // 100 "d"
		0, // 101 "e"
		0, // 102 "f"
		0, // 103 "g"
		0, // 104 "h"
		0, // 105 "i"
		0, // 106 "j"
		0, // 107 "k"
		0, // 108 "l"
		0, // 109 "m"
		0, // 110 "n"
		0, // 111 "o"
		0, // 112 "p"
		0, // 113 "q"
		0, // 114 "r"
		0, // 115 "s"
		0, // 116 "t"
		0, // 117 "u"
		0, // 118 "v"
		0, // 119 "w"
		0, // 120 "x"
		0, // 121 "y"
		0, // 122 "z"
		0, // 123 "{"
		0, // 124 "|"
		0, // 125 "}"
		0, // 126 "~"
		parseInt("101010101010101", 2) // 127 DEL
	];

	var MORSE_CODES_MAP = function (map) {
		for (var i = 0, len = MORSE_CODES.length; i < len; i++) {
			if (MORSE_CODES[i]) {
				map[ MORSE_CODES[i] ] = String.fromCharCode(i);
			}
		}
		return map;
	} ({});

	var MorseDevice = function () { this.init.apply(this, arguments) };
	MorseDevice.prototype = {
		init : function (opts) {
			var self = this;
			self.opts = opts;
			self.initListener();
			self.queue = '';
			self.buffer = '';
			self.deviceQueue = 0;

			self.addListener('sent', function (e) {
				self.deviceQueue = e.buffer;
				self.buffer = self.buffer.substring(1);
				self.dispatchEvent('buffer', { value : self.buffer });
				self.getDeviceBuffer(function (deviceBuffer) {
					self.buffer = deviceBuffer;
					self.dispatchEvent('buffer', { value : deviceBuffer });
				});
				self.exhaust();
			});
		},

		connect : function (callback) {
			var self = this;
			self._requestId = 1;
			self._callbacks = {};

			self.socket = new WebSocket(self.opts.server);
			self.socket.onopen = function () {
				self.dispatchEvent('connected', {});
			};
			self.socket.onclose = function () {
				self.dispatchEvent('disconnected', {});
				delete self.socket;
				if (self.opts.autoReconnect) setTimeout(function () {
					console.log('reconnecting');
					self.connect();
				}, 1000);
			};
			self.socket.onmessage = function (e) {
				console.log(e.data);
				var data = JSON.parse(e.data);
				if (typeof data.id == 'number') {
					if (self._callbacks[data.id]) {
						self._callbacks[data.id](data);
					} else {
						console.log('unknown id response', data);
					}
				} else {
					console.log('event', data.result.event, data.result.value);
					self.dispatchEvent(data.result.event, data.result.value);
				}
			};
		},

		command : function (method, params, callback) {
			var self = this;
			var id = self._requestId++;
			self._callbacks[id] = function (data) {
				delete self._callbacks[id];
				if (data.error) {
					throw data.error;
				}
				if (callback) callback(data.result);
			};
			self.socket.send(JSON.stringify({
				method : method,
				params : params,
				id     : id
			}));
		},

		disconnect : function (callback) {
			var self = this;
			if (self.socket) {
				self.socket.close();
			}
		},

		send : function (string) {
			var self = this;
			self.queue += string;
			self.dispatchEvent('queue', { value : self.queue });
			self.exhaust();
		},

		exhaust : _.throttle(function () {
			var self = this;
			var MIN = config.queue.MIN;
			var MAX = config.queue.MAX;

			if (self._exhaust) return;
			if (!self.queue.length) return;
			if (self.deviceQueue < MIN) {
				self.getDeviceBuffer(function (deviceBuffer) {
					console.log('deviceBuffer', deviceBuffer.length);
					if (!(deviceBuffer.length < MIN)) return;
					console.log('do exhaust', self.deviceQueue, self.queue.length);
					var max = MAX - self.deviceQueue;
					self.deviceQueue += max;
					var send =  self.queue.substring(0, max);
					self.queue = self.queue.substring(max);
					self.buffer += send;
					self._exhaust = true;
					self.dispatchEvent('queue', { value : self.queue });
					self.dispatchEvent('buffer', { value : self.buffer });
					self.command('send', [ send ], function (data) {
						self._exhaust = false;
					});
				});
			} else {
				console.log('pending exhaust', self.deviceQueue, self.queue.length);
			}
		}, 100),

		getDeviceBuffer : function (callback) {
			var self = this;
			self.command('device_buffer', [], callback);
		},

		setSpeed : _.throttle(function (speed, callback) {
			var self = this;
			self.command('speed', [ speed ], callback);
		}, 250),

		getSpeed : function (callback) {
			var self = this;
			self.command('speed', [], callback);
		},

		setInhibitTime : _.throttle(function (inhibit_time, callback) {
			var self = this;
			self.command('inhibit_time', [ inhibit_time ], callback);
		}, 250),

		getInhibitTime : function (callback) {
			var self = this;
			self.command('inhibit_time', [], callback);
		},

		setTone : _.throttle(function (tone, callback) {
			var self = this;
			self.command('tone', [ tone ], callback);
		}, 250),

		getTone : function (callback) {
			var self = this;
			self.command('tone', [], callback);
		},

		stop : _.throttle(function (callback) {
			var self = this;
			self.queue = '';
			self.deviceQueue = 0;
			self.dispatchEvent('queue', { value : self.queue });
			self.command('stop', [], callback);
			self.getDeviceBuffer(function (deviceBuffer) {
				self.dispatchEvent('buffer', { value : deviceBuffer });
			});
		}, 100),

		back : _.throttle(function (callback) {
			var self = this;
			if (self.queue) {
				self.queue = self.queue.slice(0, -1);
				self.dispatchEvent('queue', { value : self.queue });
				setTimeout(callback, 0);
			} else {
				self.buffer = self.buffer.slice(0, -1);
				self.dispatchEvent('buffer', { value : self.buffer });
				self.command('back', [], callback);
			}
		}, 100),

		initListener : function () {
			this.listeners = {};
		},

		addListener : function (event, listener) {
			if (!this.listeners[event]) this.listeners[event] = [];
			this.listeners[event].push(listener);
		},

		removeListener : function (event, listener) {
			if (!this.listeners[event]) return;
			for (var i = 0, it; (it = this.listeners[event][i]); i++) {
				if (it === listener) {
					this.listeners[event].splice(i, 1);
					return;
				}
			}
		},

		dispatchEvent : function (event, data) {
			if (!this.listeners[event]) return;
			try {
				for (var i = 0, it; (it = this.listeners[event][i]); i++) {
					it(data);
				}
			} catch (e) {
				console.log(e, e.stack);
			}
		},

		encodeMorseCodeToBits : function (string) {
			var ret = 0;
			for (var i = 0, len = string.length; i < len; i++) {
				var char = string.charAt(i);
				if (char === ".") {
					ret = ret << 2 | 0x01;
				} else
				if (char === "-") {
					ret = ret << 4 | 0x07;
				}
				if (ret > 0xffffffff || ret < 0) throw "too long code";
			}
			return ret;
		}

	};
	return MorseDevice;
});

App.controller('MainCtrl', function ($scope, $timeout, $document, $modal, config, MorseDevice) {
	var device = new MorseDevice({
		server : config.server,
		autoReconnect : true
	});

	$scope.name = config.name;
	$scope.connected = false;
	$scope.speed = 20;
	$scope.inhibit_time = 20;
	$scope.tone = 600;
	$scope.toneEnabled = true;
	$scope.sent = [];
	$scope.buffer = '';
	$scope.queue = '';

	$scope.macros = [
		{
			"name" : "CQ",
			"text" : "CQ CQ DE JH1UMV JH1UMV JCC110305 PSE K"
		},
		{
			"name" : "EX",
			"text" : "EX EX EX DE JH1UMV JH1UMV JH1UMV                                            VVVVVVVVV JH1UMV"
		},
		{
			"name" : "QRL?",
			"text" : "QRL?"
		},
		{
			"name" : "QRZ?",
			"text" : "QRZ?"
		},
		{
			"name" : "BK UR",
			"text" : "BK UR 599 5NN BK"
		},
		{
			"name" : "DE",
			"text" : "DE JH1UMV"
		},
		{
			"name" : "73",
			"text" : "73 TU E E"
		}
	];

	try {
		$scope.macros = angular.copy(JSON.parse(localStorage.macros));
	} catch (e) {
		console.log('Failed to parse JSON: ' + e);
	}

	device.addListener('connect', function () {
	});

	device.addListener('opened', function () {
		console.log('opened!!!');
		device.getSpeed(function (speed) {
			console.log('getSpeed', speed);
			$scope.$evalAsync(function () {
				$scope.speed = +speed;
			});
		});
		device.getInhibitTime(function (inhibit_time) {
			console.log('inhibit_time', inhibit_time);
			$scope.$evalAsync(function () {
				$scope.inhibit_time = +inhibit_time;
			});
		});
		device.getTone(function (tone) {
			console.log('tone', tone);
			$scope.$evalAsync(function () {
				tone = +tone;
				if (tone) {
					$scope.tone = tone;
					$scope.toneEnabled = true;
				} else {
					$scope.toneEnabled = false;
				}
			});
		});

		$scope.$evalAsync(function () {
			$scope.connected = true;
		});
	});

	device.addListener('closed', function () {
		$scope.$evalAsync(function () {
			$scope.connected = false;
		});
	});

	device.addListener('disconnected', function () {
		$scope.$evalAsync(function () {
			$scope.connected = false;
		});
	});

	device.addListener('sent', function (e) {
		$scope.$evalAsync(function () {
			if (e.char) {
				$scope.sent.push(e.char);
			} else {
				$scope.sent.push('<' + e.sign.replace(/111/g, '-').replace(/1/g, '.').replace(/0/g, '') + '>');
			}
			while ($scope.sent.length > 20) $scope.sent.shift();
		});
	});

	device.addListener('buffer', function (e) {
		console.log('buffer', e);
		$scope.$evalAsync(function () {
			$scope.buffer = e.value;
		});
	});

	device.addListener('queue', function (e) {
		console.log('queue', e);
		$scope.$evalAsync(function () {
			$scope.queue = e.value;
		});
	});

	$scope.$watch('speed', function (newValue, oldValue) {
		if (newValue === oldValue) return;
		device.setSpeed(newValue, function () {
			console.log('speed updated');
		});
	});

	$scope.$watch('inhibit_time', function (newValue, oldValue) {
		if (newValue === oldValue) return;
		device.setInhibitTime(newValue, function () {
			console.log('inhibit_time updated');
		});
	});

	$scope.$watch('tone', function (newValue, oldValue) {
		if (newValue === oldValue) return;
		device.setTone(newValue, function () {
			console.log('tone updated');
		});
	});

	$scope.$watch('toneEnabled', function (newValue, oldValue) {
		if (newValue === oldValue) return;
		if (newValue) {
			device.setTone($scope.tone, function () {
				console.log('tone updated');
			});
		} else {
			device.setTone(0, function () {
				console.log('tone updated');
			});
		}
	});

	$scope.send = function (string) {
		device.send(string + ' ');
		$document.find('#input').focus();
	};

	$scope.edit = function (macro) {
		var modal = $modal.open({
			templateUrl: "edit.html",
			controller : function ($scope, $modalInstance, macro) {
				$scope.mode  = macro ? 'edit' : 'add';
				$scope.macro = $scope.mode === 'edit' ? angular.copy(macro) : {};

				$scope.ok = function () {
					$modalInstance.close(angular.copy($scope.macro));
				};

				$scope.cancel = function () {
					$modalInstance.dismiss('cancel');
				};

				$scope.remove = function () {
					if (confirm("Sure?")) {
						$modalInstance.close('remove');
					}
				};
			},
			resolve : {
				macro : function () {
					return macro;
				}
			}
		});

		modal.result.then(function (m) {
			if (m != 'remove') {
				console.log(m);
				if (macro) {
					macro.name = m.name;
					macro.text = m.text;
				} else {
					$scope.macros.push(m);
				}
			} else {
				for (var i = 0, len = $scope.macros.length; i < len; i++) {
					if ($scope.macros[i] === macro) {
						$scope.macros.splice(i, 1);
						break;
					}
				}
			}

			localStorage.macros = JSON.stringify(angular.copy($scope.macros));
		});
	};

	var input = $document.find('#input');
	input.keydown(function (e) {
		var key = keyString(e.originalEvent);
		console.log(key);

		if (key === 'bf') key = '/';
		if (key === 'bb') key = '+';
		if (key === 'S-bb') key = '=';
		if (key === 'S-bf') key = '?';
		if (key === 'S-BS') key = '\x7f';

		if (/^[a-z0-9=+\-\?\/\x7f]$/i.test(key)) {
			device.send(key);
		} else
		if ('SPC' === key) {
			device.send(' ');
		} else
		if ('ESC' === key || 'C-C' === key) {
			device.stop();
		} else
		if ('BS' === key) {
			device.back();
		} else
		if ('C-RET' === key) { // BT
			device.send('=');
		} else
		if ('S-C-RET' === key) { // AR
			device.send('+');
		}

		return false;
	});
	input.focus();

	device.connect();
});



