// NOTE: this is copy-pasted from `auth/public/login.js` and should 
// eventually be shared in one place or deleted.

function setStatusInfo(message) {
  var statusText = document.getElementById("statusText")
  statusText.innerHTML = message
  statusText.classList.remove("statusError")
  statusText.classList.add("statusInfo")
}

function setStatusError(message) {
  var statusText = document.getElementById("statusText")
  statusText.innerHTML = message
  statusText.classList.remove("statusInfo")
  statusText.classList.add("statusError")
}

function clearStatusText() {
  var statusText = document.getElementById("statusText")
  statusText.innerHTML = ""
  statusText.classList.remove("statusInfo")
  statusText.classList.remove("statusError")
}

function onFieldBlur(event) {
  var hasValue = !!event.target.value
  if (hasValue) {
    event.target.parentNode.classList.add("noPlaceholder")
  } else {
    event.target.parentNode.classList.remove("noPlaceholder")
  }
}

function onFieldFocus(event) {
  event.target.parentNode.classList.add("noPlaceholder")
  clearStatusText()
}

function onCreateAcct(event) {
  setStatusError("Account creation not yet implemented.")
}

function onForgotCred(event) {
  setStatusError("Forgotten credentials not yet implemented.")
}

function onSocialLogin(type, event) {
  event.preventDefault()
  setStatusError("Social login not yet implemented.")
}
