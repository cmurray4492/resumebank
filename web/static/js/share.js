// Event-delegated "copy link" handler for any .js-copy-link button (used by
// the share_buttons partial on job and blog pages). Delegated on document
// so it works for buttons added after this script runs, with no per-page setup.
document.addEventListener('click', function (event) {
  var button = event.target.closest('.js-copy-link');
  if (!button) return;

  var url = button.getAttribute('data-share-url');
  if (!url || !navigator.clipboard) return;

  navigator.clipboard.writeText(url).then(function () {
    var original = button.textContent;
    button.textContent = 'Copied!';
    button.disabled = true;
    setTimeout(function () {
      button.textContent = original;
      button.disabled = false;
    }, 1500);
  }).catch(function () {
    // Clipboard access can be denied (e.g. insecure context, permissions) -
    // the other share links still work, so just leave the button as-is.
  });
});
