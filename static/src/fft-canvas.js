Polymer({
	is: 'my-fft-canvas',
	properties: {
		fftSize : {
			type: Number,
			value: 0
		},

		initialized: {
			type: Boolean,
			value: false
		}
	},

	created: function () {
	},

	ready: function () {
		var self = this;
	},

	attached : function () {
		var self = this;
	},

	init : function () {
		var self = this;

		self.$.fftCanvas.height = 100;
		self.$.fftCanvas.width = self.offsetWidth * (window.devicePixelRatio || 1);

		self.set('initialized', true);
	},

	render : function (array) {
		var self = this;
		var width = self.$.fftCanvas.width;
		var height = self.$.fftCanvas.height;

		var min = Math.ceil(array.length / width);

		var ctx = self.$.fftCanvas.getContext('2d');
		ctx.fillStyle = "#000000";
		ctx.lineWidth = 1;

		cancelAnimationFrame(self._requestedFFT);
		self._requestedFFT = requestAnimationFrame(function () {

			// ctx.fillRect(0, 0, width, height);
			self.$.fftCanvas.width = self.$.fftCanvas.width;

			// draw grid
			ctx.strokeStyle = "#666666";
			ctx.beginPath();
			ctx.moveTo(width / 2, 0);
			ctx.lineTo(width / 2, height);
			ctx.stroke();

			// draw FFT result
			ctx.strokeStyle = "#ffffff";
			ctx.beginPath();
			ctx.moveTo(0, height);
			for (var i = 0, len = width; i < len; i += 2) {
				var n = ~~(i / width * array.length);
				var v = array[n];
				// var v = array.subarray(n, n + min).reduce(function (i, r) { return i + r }) / min;
				var p = v / 80;
				ctx.lineTo(~~(i), ~~(height - (p * height)));
			}
			ctx.stroke();
		});
	}
});



