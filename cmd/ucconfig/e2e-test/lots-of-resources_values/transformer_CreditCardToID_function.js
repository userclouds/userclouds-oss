
function id(len, decimalonly) {
	var s = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";
	var d = "0123456789";

	if (decimalonly) {
		return Array(len).join().split(',').map(function() {
			return d.charAt(Math.floor(Math.random() * d.length));
		}).join('');
	}

	return Array(len).join().split(',').map(function() {
		return s.charAt(Math.floor(Math.random() * s.length));
	}).join('');
}

function constructSegment(seg, decimalonly, preserveS, preserveT) {
	preserveCountS = Math.min(Math.max(preserveS, 0), seg.length);
	preserveCountT = Math.min(Math.max(preserveT, 0), seg.length);

	preserveCount = preserveCountS + preserveCountT
	if (preserveCount >= seg.length) {
		return seg
	}

	newSegS = seg.slice(0, preserveCountS)
	newSegT = seg.slice(seg.length - preserveCountT, seg.length)
	return newSegS + id(seg.length - preserveCount, decimalonly) + newSegT;
}

function validate(str) {
	// there a lot of different regexes per type of card
	// for now just test the length
	return (str.length == 16);
}

function transform(data, params) {
	// Strip non numeric characters if present
	orig_data = data;
	data = data.replace(/\D/g, '');
	if (!validate(data)) {
		throw new Error('Invalid Credit Card Number Provided');
	}

	if ((params.PreserveCharsTrailing + params.PreserveCharsStart) > 9 ||
		params.PreserveCharsTrailing < 0 ||
		params.PreserveCharsStart < 0) {
		throw new Error('Invalid Params Provided');
	}

	if (params.PreserveValue) {
		return orig_data;
	}

	seg1 = data.slice(0, 4);
	seg2 = data.slice(4, 8);
	seg3 = data.slice(8, 12);
	seg4 = data.slice(12, 16);
	return constructSegment(
			seg1,
			params.DecimalOnly,
			params.PreserveCharsStart,
			params.PreserveCharsTrailing - 12
		) +
		'-' +
		constructSegment(
			seg2,
			params.DecimalOnly,
			params.PreserveCharsStart - 4,
			params.PreserveCharsTrailing - 8
		) +
		'-' +
		constructSegment(seg3,
			params.DecimalOnly,
			params.PreserveCharsStart - 8,
			params.PreserveCharsTrailing - 4
		) +
			'-' +
		constructSegment(
			seg4,
			params.DecimalOnly,
			params.PreserveCharsStart - 12,
			params.PreserveCharsTrailing
		);
};
