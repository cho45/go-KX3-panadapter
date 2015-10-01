Polymer({
	is: 'my-fft-canvas',
	properties: {
		height : {
			type: Number,
			value: 100
		},

		width : {
			type: Number,
			value: 0
		},

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

		self.$.fftCanvas.height = self.height;
		self.$.fftCanvas.width = self.width || self.offsetWidth * (window.devicePixelRatio || 1);

		self.initWebGL();
		self.set('initialized', true);
	},

	initWebGL : function () {
		var self = this;

		try {
			self.gl = self.$.fftCanvas.getContext("webgl") || self.$.fftCanvas.getContext("experimental-webgl");
		} catch (e) {
			console.log(e);
		}

		if (!self.gl) {
			alert("Unable to initialize WebGL. Your browser may not support it.");
			return;
		}

		var gl = self.gl;

		gl.disable(gl.DEPTH_TEST);
		gl.disable(gl.CULL_FACE);
		gl.disable(gl.BLEND);

		gl.viewport(0, 0, self.$.fftCanvas.width, self.$.fftCanvas.height);
		gl.clearColor(0.0, 0.0, 0.0, 1.0);
		gl.clear(gl.COLOR_BUFFER_BIT);

		var fragmentShader = gl.createShader(gl.FRAGMENT_SHADER);
		gl.shaderSource(fragmentShader, self.$.fragmentShader.textContent);
		gl.compileShader(fragmentShader);
		if (!gl.getShaderParameter(fragmentShader, gl.COMPILE_STATUS)) {  
			alert("An error occurred compiling the shaders: " + gl.getShaderInfoLog(fragmentShader));  
			return;
		}

		var vertexShader = gl.createShader(gl.VERTEX_SHADER);
		gl.shaderSource(vertexShader, self.$.vertexShader.textContent);
		gl.compileShader(vertexShader);
		if (!gl.getShaderParameter(vertexShader, gl.COMPILE_STATUS)) {  
			alert("An error occurred compiling the shaders: " + gl.getShaderInfoLog(vertexShader));  
			return;
		}

		self.shaderProgram = gl.createProgram();
		gl.attachShader(self.shaderProgram, vertexShader);
		gl.attachShader(self.shaderProgram, fragmentShader);
		gl.linkProgram(self.shaderProgram);

		if (!gl.getProgramParameter(self.shaderProgram, gl.LINK_STATUS)) {
			alert("Unable to initialize the shader program.");
		}

		gl.useProgram(self.shaderProgram);

		self.vertexPositionAttribute = gl.getAttribLocation(self.shaderProgram, "aVertexPosition");
		gl.enableVertexAttribArray(self.vertexPositionAttribute);

		gl.uniformMatrix4fv(gl.getUniformLocation(self.shaderProgram, 'uPMatrix'), false, new Float32Array([
			 2,  0,  0,  0,
			 0,  2,  0,  0,
			 0,  0,  1,  0,
			-1, -1,  0,  1
		]));

		self.verticesBuffer = new Float32Array(self.$.fftCanvas.width * 2);

		self.vertices1 = gl.createBuffer();
		gl.bindBuffer(gl.ARRAY_BUFFER, self.vertices1);
		gl.vertexAttribPointer(self.vertexPositionAttribute, 2, gl.FLOAT, false, 0, 0);

		gl.lineWidth(window.devicePixelRatio || 1);
	},

	render : function (array) {
		var self = this;
		var gl = self.gl;

		var width = self.$.fftCanvas.width;
		var height = self.$.fftCanvas.height;

		gl.clear(gl.COLOR_BUFFER_BIT);

		var buffer = self.verticesBuffer;
		for (var i = 0, len = width; i < len; i++) {
			var n = ~~(i / width * array.length);
			var v = array[n];
			var p = v / 80;
			buffer[i*2+0] = i / width;
			buffer[i*2+1] = p;
		}

		gl.bufferData(gl.ARRAY_BUFFER, buffer, gl.DYNAMIC_DRAW);
		gl.drawArrays(gl.LINE_STRIP, 0, width);
	}

//	render : function (array) {
//		var self = this;
//		var width = self.$.fftCanvas.width;
//		var height = self.$.fftCanvas.height;
//
//		var min = Math.ceil(array.length / width);
//
//		var ctx = self.$.fftCanvas.getContext('2d');
//		ctx.fillStyle = "#000000";
//		ctx.lineWidth = 1;
//
//		cancelAnimationFrame(self._requestedFFT);
//		self._requestedFFT = requestAnimationFrame(function () {
//
//			// ctx.fillRect(0, 0, width, height);
//			self.$.fftCanvas.width = self.$.fftCanvas.width;
//
//			// draw grid
//			ctx.strokeStyle = "#666666";
//			ctx.beginPath();
//			ctx.moveTo(width / 2, 0);
//			ctx.lineTo(width / 2, height);
//			ctx.stroke();
//
//			// draw FFT result
//			ctx.strokeStyle = "#ffffff";
//			ctx.beginPath();
//			ctx.moveTo(0, height);
//			for (var i = 0, len = width; i < len; i += 2) {
//				var n = ~~(i / width * array.length);
//				var v = array[n];
//				// var v = array.subarray(n, n + min).reduce(function (i, r) { return i + r }) / min;
//				var p = v / 80;
//				ctx.lineTo(~~(i), ~~(height - (p * height)));
//			}
//			ctx.stroke();
//		});
//	}
});



