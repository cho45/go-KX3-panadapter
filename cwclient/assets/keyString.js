function keyString (e) {
	var ret = '';
	var key = e.keyCode || e.which;
	if (e.shiftKey) ret += 'S-';
	if (e.ctrlKey) ret += 'C-';
	if (e.altKey)  ret += 'M-';
	if (e.metaKey && !e.ctrlKey) ret += 'W-';
	if (48 <= key && key <= 90) {
		ret += String.fromCharCode(key);
	} else {
		ret += arguments.callee.table1[key] || key.toString(16);
	}
	return ret;
}
keyString.table1 = { 8 : "BS", 13 : "RET", 32 : "SPC", 9 : "TAB", 27 : "ESC", 33 : "PageUp", 34 : "PageDown", 35 : "End", 36 : "Home", 37 : "Left", 38 : "Up", 39 : "Right", 40 : "Down", 45 : "Insert", 46 : "Delete", 112 : "F1", 113 : "F2", 114 : "F3", 115 : "F4", 116 : "F5", 117 : "F6", 118 : "F7", 119 : "F8", 120 : "F9", 121 : "F10", 122 : "F11", 123 : "F12" };
