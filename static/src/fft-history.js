Polymer({
	is: 'my-fft-history',
	properties: {
		fftSize : {
			type: Number,
			value: 0
		},

		historySize : {
			type: Number,
			value: 512
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
		self._current = 0;

		self.$.historyCanvas.height = self.historySize;
		self.$.historyCanvas.width  = self.offsetWidth * (window.devicePixelRatio || 1);

		self.initWebGL();
		self.set('initialized', true);
	},

	initWebGL : function () {
		var self = this;

		try {
			self.gl = self.$.historyCanvas.getContext("webgl") || self.$.historyCanvas.getContext("experimental-webgl");
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

		gl.viewport(0, 0, self.$.historyCanvas.width, self.$.historyCanvas.height);
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

		self.vertices1 = gl.createBuffer();
		gl.bindBuffer(gl.ARRAY_BUFFER, self.vertices1);
		gl.bufferData(gl.ARRAY_BUFFER, new Float32Array([
			1.0,  1.0,  0.0,
			-1.0, 1.0,  0.0,
			1.0,  -1.0, 0.0,
			-1.0, -1.0, 0.0
		]), gl.STATIC_DRAW);

		// texture sources
		self.textures = [gl.createTexture(), gl.createTexture()];

		// just for initializing
		var width = self.fftSize;
		var height = self.historySize;
		var array = new Uint8Array(width * height * 4);

		for (var i = 0, it; (it = self.textures[i]); i++) {
			gl.bindTexture(gl.TEXTURE_2D, it);
			gl.pixelStorei(gl.UNPACK_COLORSPACE_CONVERSION_WEBGL, gl.NONE);
			gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, width, height, 0, gl.RGBA, gl.UNSIGNED_BYTE, array);
			gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST);
			gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST);
			gl.bindTexture(gl.TEXTURE_2D, null);
		}

		self.lineBuffer = new Uint8Array(self.fftSize * 4);

		gl.uniform2f(gl.getUniformLocation(self.shaderProgram, 'uViewCoords'), self.$.historyCanvas.width, self.$.historyCanvas.height);

		gl.bindBuffer(gl.ARRAY_BUFFER, self.vertices1);
		gl.vertexAttribPointer(self.vertexPositionAttribute, 3, gl.FLOAT, false, 0, 0);

		gl.activeTexture(gl.TEXTURE1);
		gl.bindTexture(gl.TEXTURE_2D, self.textures[1]);
		gl.uniform1i(gl.getUniformLocation(self.shaderProgram, "uTexture1"), 1);

		gl.activeTexture(gl.TEXTURE0);
		gl.bindTexture(gl.TEXTURE_2D, self.textures[0]);
		gl.uniform1i(gl.getUniformLocation(self.shaderProgram, "uTexture0"), 0);

		gl.bindTexture(gl.TEXTURE_2D, self.textures[0]);

		self._renderHistory();
	},

	render : function (array) {
		var self = this;
		if (!self.initialized) return;

		var gl = self.gl;

		var data = self.lineBuffer;

		for (var i = 0, len = self.fftSize; i < len; i++) {
			var n = i * 4;
			var r = 0, g = 0, b = 0;
			var p = array[i] / 80;

			if (p < 1.0/6.0) {
				// black -> blue
				p = p / (1 / 6.0);
				r = 0;
				g = 0;
				b = 255 * p;
			} else
			if (p < 2.0/6.0) {
				// blue -> light blue
				p = (p - (1 / 6.0)) / (1 / 6.0);
				r = 0;
				g = 255 * p;
				b = 255;
			} else
			if (p < 3.0/6.0) {
				// light blue -> green
				p = (p - (2 / 6.0)) / (1 / 6.0);
				r = 0;
				g = 255;
				b = 255 * (1 - p);
			} else
			if (p < 4.0/6.0) {
				// green -> yellow
				p = (p - (3 / 6.0)) / (1 / 6.0);
				r = 255 * p;
				g = 255;
				b = 0;
			} else
			if (p < 5.0/6.0) {
				// yellow -> red
				p = (p - (4 / 6.0)) / (1 / 6.0);
				r = 255;
				g = 255 * (1 - p);
				b = 0;
			} else {
				// yellow -> red
				p = (p - (5 / 6.0)) / (1 / 6.0);
				r = 255;
				g = 255 * p;
				b = 255 * p;
			}

			data[n + 0] = r;
			data[n + 1] = g;
			data[n + 2] = b;
			data[n + 3] = 255;
		}

		var xoffset = 0, yoffset = self._current, width = self.fftSize, height = 1;
		gl.texSubImage2D(gl.TEXTURE_2D, 0, xoffset, yoffset, width, height, gl.RGBA, gl.UNSIGNED_BYTE, data);

		self._current++;

		if (self._current >= self.historySize) {
			self._current = 0;
			self.textures.push(self.textures.shift());

			gl.activeTexture(gl.TEXTURE1);
			gl.bindTexture(gl.TEXTURE_2D, self.textures[1]);
			gl.uniform1i(gl.getUniformLocation(self.shaderProgram, "uTexture1"), 1);

			gl.activeTexture(gl.TEXTURE0);
			gl.bindTexture(gl.TEXTURE_2D, self.textures[0]);
			gl.uniform1i(gl.getUniformLocation(self.shaderProgram, "uTexture0"), 0);

		}

		cancelAnimationFrame(self._requestedHistory);
		self._requestedHistory = requestAnimationFrame(function () {
			self._renderHistory();
		});
	},

	shiftFFTHistory : function (shift) {
		var self = this;
		if (!self.initialized) return;

		var gl = self.gl;

		var width = self.fftSize;
		var height = self.historySize;

		var array = new Uint8Array(width * height * 4);

		if (Math.abs(shift) > (self.fftSize / 2)) {
			// just clear texture
			for (var i = 0, texture; (texture = self.textures[i]); i++) {
				gl.bindTexture(gl.TEXTURE_2D, texture);
				gl.texSubImage2D(gl.TEXTURE_2D, 0, 0, 0, width, height, gl.RGBA, gl.UNSIGNED_BYTE, array);
				gl.bindTexture(gl.TEXTURE_2D, null);
			}
		} else {
			for (var ti = 0, texture; (texture = self.textures[ti]); ti++) { // no warnings
				// gl.activeTexture(i === 1 ? gl.TEXTURE1 : gl.TEXTURE0);

				var fb = gl.createFramebuffer();
				gl.bindFramebuffer(gl.FRAMEBUFFER, fb);
				gl.framebufferTexture2D( gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, texture, 0);
				gl.readPixels(0, 0, width, height, gl.RGBA, gl.UNSIGNED_BYTE, array);
				gl.bindFramebuffer(gl.FRAMEBUFFER, null);

				if (shift < 0) {
					// shift to left and fill black to right
					for (var y = 0; y < height; y++) {
						for (var i = width - 1; -shift < i; i--) {
							array[(y * width + i) * 4 + 0] = array[(y * width + i + shift) * 4 + 0];
							array[(y * width + i) * 4 + 1] = array[(y * width + i + shift) * 4 + 1];
							array[(y * width + i) * 4 + 2] = array[(y * width + i + shift) * 4 + 2];
							array[(y * width + i) * 4 + 3] = array[(y * width + i + shift) * 4 + 3];
						}
						for (var i = 0; i < -shift; i++) {
							array[(y * width + i) * 4 + 0] = 0;
							array[(y * width + i) * 4 + 1] = 0;
							array[(y * width + i) * 4 + 2] = 0;
							array[(y * width + i) * 4 + 3] = 255;
						}
					}
				} else {
					// shift to right and fill black to left
					for (var y = 0; y < height; y++) { // no warnings
						for (var i = 0; i < width - shift; i++) {
							array[(y * width + i) * 4 + 0] = array[(y * width + i + shift) * 4 + 0];
							array[(y * width + i) * 4 + 1] = array[(y * width + i + shift) * 4 + 1];
							array[(y * width + i) * 4 + 2] = array[(y * width + i + shift) * 4 + 2];
							array[(y * width + i) * 4 + 3] = array[(y * width + i + shift) * 4 + 3];
						}
						for (var i = width - shift; i < width; i++) {
							array[(y * width + i) * 4 + 0] = 0;
							array[(y * width + i) * 4 + 1] = 0;
							array[(y * width + i) * 4 + 2] = 0;
							array[(y * width + i) * 4 + 3] = 255;
						}
					}
				}

				gl.bindTexture(gl.TEXTURE_2D, texture);
				gl.texSubImage2D(gl.TEXTURE_2D, 0, 0, 0, width, height, gl.RGBA, gl.UNSIGNED_BYTE, array);
				gl.bindTexture(gl.TEXTURE_2D, null);
			}
		}

		gl.activeTexture(gl.TEXTURE1);
		gl.bindTexture(gl.TEXTURE_2D, self.textures[1]);
		gl.uniform1i(gl.getUniformLocation(self.shaderProgram, "uTexture1"), 1);

		gl.activeTexture(gl.TEXTURE0);
		gl.bindTexture(gl.TEXTURE_2D, self.textures[0]);
		gl.uniform1i(gl.getUniformLocation(self.shaderProgram, "uTexture0"), 0);
	},

	_renderHistory : function () {
		var self = this;
		var gl = self.gl;

		gl.uniform1f(gl.getUniformLocation(self.shaderProgram, 'uOffsetY'), self._current);

		gl.drawArrays(gl.TRIANGLE_STRIP, 0, 4);
	}
});


