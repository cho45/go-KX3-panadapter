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
		},

		buffer: {
			type: String,
			value: ""
		},

		queue : {
			type: String,
			value: ""
		},

		sent : {
			type: Array,
			value: []
		},

		sendingState : {
			type: String,
			value : ""
		},

		textHistory: {
			type: Array,
			value: []
		},

		textHistoryIndex: {
			type: Number,
			value: 0
		},

		textInputShow : {
			type: Boolean,
			value: false,
			observer: "_textInputShowChanged"
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
		self.openMorseDevice();
	},

	initCanvas : function () {
		var self = this;
		self._current = 0;

		self.$.fftHistory.fftSize = self.config.fftSize;
		self.$.fftCanvas.fftSize = self.config.fftSize;

		navigator.getBattery().then(function (bm) {
			self.$.fftHistory.width = self.$.fftCanvas.width = (bm.charging ? 0 : self.offsetWidth);

			self.$.fftHistory.init();
			self.$.fftCanvas.init();
		});
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

		window.addEventListener('keydown', function (e) {
			var key = (e.altKey?"Alt-":"")+(e.ctrlKey?"Control-":"")+(e.metaKey?"Meta-":"")+(e.shiftKey?"Shift-":"")+e.key;
			console.log('window.onkeydown', key);

			if (key === 'Enter') {
				e.preventDefault();
				self.textInputShow = !self.textInputShow;
			} else
			if (key === 'Escape') {
				e.preventDefault();
				self.textInputShow = false;
				self.device.stop();
			} else
			if (key === 'Backspace') {
				e.preventDefault();
			}
		});

		self.$.textInputInput.addEventListener('keydown', function (e) {
			e.stopPropagation();
			var key = (e.altKey?"Alt-":"")+(e.ctrlKey?"Control-":"")+(e.metaKey?"Meta-":"")+(e.shiftKey?"Shift-":"")+e.key;
			console.log('input.onkeydown', key);

			if (/^(?:Shift-)?([A-Za-z0-9=+\-\?\/&\*%!\( ])$/.test(key)) {
				if (self.$.textInputImmdiately.checked) {
					var key = RegExp.$1;
					e.preventDefault();
					self.device.send(key.toUpperCase());
				} else {
					// do nothing
				}
			} else
			if ('Escape' === key || 'Control-c' === key) {
				e.preventDefault();
				self.device.stop();
				self.textInputShow = false;
				self.$.textInputInput.inputElement.blur();
			} else
			if ('Enter' === key) {
				e.preventDefault();
				var text = self.$.textInputInput.value.toUpperCase();
				if (text) {
					console.log('SENDING', text);
					self.$.textInputInput.value = '';
					self.device.send(text);

					if (self.textHistory[ self.textHistory.length - 1] !== text) {
						self.push('textHistory', text);
						while (self.textHistory.length > 100) self.shift('textHistory');
					}
					self.textHistoryIndex = 0;
				}
			} else
			if ('Control-Enter' === key) { // BT
				e.preventDefault();
				self.device.send('=');
			} else
			if ('Shift-Control-Enter' === key) { // AR
				e.preventDefault();
				self.device.send('+');
			} else
			if ('ArrowUp' === key || "Control-p" === key) {
				e.preventDefault();
				var history = self.textHistory.slice(0).reverse();
				self.textHistoryIndex++;
				if (self.textHistoryIndex > history.length) {
					self.textHistoryIndex = history.length;
				}
				try {
					self.$.textInputInput.value = history[self.textHistoryIndex-1];
				} catch (e) { }
			} else
			if ('ArrowDown' === key || "Control-n" === key) {
				e.preventDefault();
				var history = self.textHistory.slice(0).reverse(); // no warnings
				if (self.textHistoryIndex > 0) self.textHistoryIndex--;
				try {
					self.$.textInputInput.value = history[self.textHistoryIndex-1] || "";
				} catch (e) { }
			} else
			if ('Tab' === key) {
				e.preventDefault();
				self.$.textInputInput.value = '';
				self.$.textInputImmdiately.checked = !self.$.textInputImmdiately.checked;
				self.async(function () {
					self.$.textInputInput.inputElement.focus();
				}, 10);
			} else
			if ('Backspace' === key) {
				if (!self.$.textInputInput.value.length) {
					e.preventDefault();
					self.device.back();
				} else {
				}
			} else {
				e.preventDefault();
			}
		});

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

	openMorseDevice : function () {
		var self = this;
		self.device = new MorseDevice({
			server : 'ws://localhost:51235/stream',
			autoReconnect : true,
			MIN : 5,
			MAX : 9
		});

		var timer;

		self.device.addListener('sent', function (e) {
			if (e.char) {
				self.push('sent', e.char);
			} else {
				self.push('sent', '<' + e.sign.replace(/111/g, '-').replace(/1/g, '.').replace(/0/g, '') + '>');
			}
			while (self.sent.length > 30) self.shift('sent');

			activate();
		});

		self.device.addListener('buffer', function (e) {
			self.set('buffer', e.value);
			activate();
		});

		self.device.addListener('queue', function (e) {
			self.set('queue', e.value);
			activate();
		});

		self.device.connect();

		function activate () {
			self.cancelAsync(timer);
			if (!self.buffer && !self.queue) {
				// all data is sent
				timer = self.async(function () {
					self.set('sent', []);
					self.toggleClass('active', false, self.$.text);
				}, 5000);
			} else {
				self.toggleClass('active', true, self.$.text);
			}
		}
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

	formatFrequency : function (frequency) {
		return String(frequency).replace(/(\d)(?=(\d{3})+(?!\d))/g, '$1,').replace(/,/, '.').slice(0, -1);
	},

	openTextInput : function () {
		this.set('textInputShow', true);
	},

	_textInputShowChanged : function () {
		var self = this;
		self.toggleClass('show', self.textInputShow, self.$.textInput);
		if (self.textInputShow) {
			self.async(function () {
				self.$.textInputInput.inputElement.focus();
			});
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

