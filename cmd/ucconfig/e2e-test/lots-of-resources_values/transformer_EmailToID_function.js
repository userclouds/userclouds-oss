function id(len) {
	var s = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";

	return Array(len).join().split(',').map(function() {
		return s.charAt(Math.floor(Math.random() * s.length));
	}).join('');
}

var commonValues = ["gmail", "hotmail", "yahoo", "msn", "aol", "orange", "wanadoo", "comcast", "live", "apple", "proton", "yandex", "ymail"]

function constructSegment(seg, config) {
	if (config.PreserveValue) {
		return seg
	}

	if (config.PreserveCommonValue && (commonValues.includes(seg))) {
		return seg
	}

	preserveCount = Math.min(config.PreserveChars, seg.length);
	newSeg = seg.slice(0, preserveCount)
	return newSeg + id(config.FinalLength - preserveCount)
}

function transform(data, params) {
	emailParts = data.split('@')

	// Make sure we have a username and a domain
	if (emailParts.length !== 2) {
		throw new Error('Invalid Data');
	}

	username = emailParts[0]
	domainParts = emailParts[1].split('.')

	// Check if the domain is valid
	if (domainParts.length < 2) {
		throw new Error('Invalid Data');
	}
	domainName = domainParts[0]
	domainExt = domainParts[1]

	if (params.length != 3) {
		throw new Error('Invalid Params');
	}
	return constructSegment(username, params[0]) + '@' +
		constructSegment(domainName, params[1]) + '.' +
		constructSegment(domainExt, params[2]);
};
