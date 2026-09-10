// Shows the strictly-necessary-cookies notice once per browser until
// dismissed. Wrapped in try/catch since localStorage can throw (private
// browsing, blocked site data) - the notice just stays visible in that case.
(function () {
  var STORAGE_KEY = 'resumebank_cookie_notice_dismissed';
  var notice = document.getElementById('cookie-notice');
  if (!notice) return;

  var dismissed = false;
  try {
    dismissed = localStorage.getItem(STORAGE_KEY) === '1';
  } catch (e) {}

  if (!dismissed) {
    notice.classList.remove('d-none');
  }

  var dismissButton = document.getElementById('cookie-notice-dismiss');
  if (dismissButton) {
    dismissButton.addEventListener('click', function () {
      notice.classList.add('d-none');
      try {
        localStorage.setItem(STORAGE_KEY, '1');
      } catch (e) {}
    });
  }
})();
