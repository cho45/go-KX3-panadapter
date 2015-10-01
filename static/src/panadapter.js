Polymer({
	is: 'my-panadapter',
	properties: {
		rigFrequency: {
			type: Number,
			value: 0
		},

		rigModeRaw: {
			type: String,
			value: ""
		},

		rigMode: {
			type: String,
			computed: "_convertRigMode(rigModeRaw)"
		},

		decodedText: {
			type: String,
			value: ""
		}
	},

	created: function () {
		var self = this;
		self.BYTE_ORDER = (function () {
			var buf = new ArrayBuffer(2);
			var i8 = new Uint8Array(buf);
			i8[0] = 0xfe;
			i8[1] = 0xff;
			var i16 = new Uint16Array(buf);
			return (i16[0] === 0xfeff) ? 'BIG_ENDIAN' : 'LITTLE_ENDIAN';
		})();

		self._id = 1;
		self._callbacks = {};
	},

	ready: function () {
		var self = this;
		console.log('ready');
	},

	attached : function () {
		var self = this;
		console.log('attached');
		self.openWebSocket();
	},

	initCanvas : function () {
		var self = this;
		self._current = 0;

		self.$.fftHistory.fftSize = self.config.fftSize;
		self.$.fftCanvas.fftSize = self.config.fftSize;

		self.$.fftHistory.init();
		self.$.fftCanvas.init();

		self.bindEvents();
	},

	bindEvents : function () {
		var self = this;
		self.$.historyContainer.onmousemove = function (e) {
		};

		self.$.historyContainer.onmousedown = function (e) {
			var freq = getFrequency(e);

			var offset = {
				CW_REV: -600,
				CW: 600
			}[self.rigMode];

			if (offset) {
				freq += offset;
			}

			console.log('change frequency to', freq, self.rigMode, 'offset', offset);
			self.request('frequency', {
				frequency: freq
			}).then(function (result) {
				console.log('change frequency result', result);
			});
		};


		window.onresize = function () {
			self.debounce('onresize', function () {
				console.log('resize');
				self.$.fftHistory.init();
				self.$.fftCanvas.init();
			}, 500);
		};

		function getFrequency(e) {
			var bcr = self.$.historyContainer.getBoundingClientRect();
			var x = e.pageX - bcr.left, y = e.pageY - bcr.top;
			// normalize to -0.5 0.5
			var pos = x / (bcr.right - bcr.left) - 0.5;
			return (self.config.input.samplerate * pos) + self.rigFrequency;
		}
	},

	openWebSocket : function () {
		var self = this;

		// TODO
		var rateLimit = location.search.match(/rate=(\d)/);
		if (rateLimit) {
			rateLimit = +rateLimit[1];
		} else {
			rateLimit = 24;
		}

		self.ws = new WebSocket('ws://localhost:51235/stream');
		self.ws.binaryType = 'arraybuffer';
		self.ws.onopen = function () {
			console.log('onopen');

			self.request('init', {
				byteOrder: self.BYTE_ORDER,
				rateLimit: rateLimit
			}).then(function (result) {
				console.log('init', result);
				self.config = result.config;
				self.rigFrequency = result.rigFrequency;
				self.rigModeRaw = result.rigMode;
				self.initCanvas();
			});
		};
		self.ws.onclose = function () {
			console.log('onclose');
		};
		self.ws.onerror = function (e) {
			console.log('onerror', e);
		};

		self.ws.onmessage = function (e) {
			if (typeof e.data === 'string') {
				var res = JSON.parse(e.data);
				if (res.id) {
					var callback = self._callbacks[res.id];
					if (!callback) {
						console.log('unknwon callback id:', res.id, self._callbacks);
					}
					if (res.error) {
						callback.reject(res.error);
					} else {
						callback.resolve(res.result);
					}
					delete self._callbacks[res.id];
				} else {
					if (res.error) {
						self.set('error', [res.error.code, res.error.message, res.error.data].join(' : '));
					} else {
						self.processNotification(res.result);
					}
				}
			} else {
				var array = new Float32Array(e.data);
				self.$.fftCanvas.render(array);
				self.$.fftHistory.render(array);
				prevRenderedTime = new Date().getTime();
			}
		};
	},

	request : function (method, params) {
		var self = this;

		return new Promise(function (resolve, reject) {
			var id = self._id++;
			self._callbacks[id] = {
				resolve: resolve,
				reject: reject
			};
			self.ws.send(JSON.stringify({
				id: id,
				method: method,
				params: params
			}));
		});
	},

	processNotification : function (result) {
		var self = this;
		if (result.type === 'frequencyChanged') {
			var freqDiff = self.rigFrequency - result.data.rigFrequency;
			self.set('rigFrequency', result.data.rigFrequency);

			var freqRes = self.config.input.samplerate / self.config.fftSize;
			var shift = Math.round(freqDiff / freqRes);

			self.$.fftHistory.shiftFFTHistory(-shift);
		} else
		if (result.type === 'modeChanged') {
			self.set('rigModeRaw', result.data.rigMode);
		} else
		if (result.type === 'decoded') {
			self.set('decodedText', (self.decodedText + result.data.char).slice(-40));
		} else {
			console.log('unexpected notification', result);
		}
	},

	_convertRigMode : function (raw) {
		return ({
			"1" : "LSB",
			"2" : "USB",
			"3" : "CW",
			"4" : "FM",
			"5" : "AM",
			"6" : "DATA",
			"7" : "CW_REV",
			"8" : "DATA_REV"
		})[raw];
	}
});

