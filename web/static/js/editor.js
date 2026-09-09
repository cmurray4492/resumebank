// Wires a Quill "snow" editor into containerId, seeding it from the current
// value of the hidden textarea hiddenInputId (server-rendered as escaped
// element content, so no HTML needs to be passed through a JS string
// literal), and copying its HTML back into that textarea on form submit.
function initRichTextEditor(containerId, hiddenInputId) {
  var container = document.getElementById(containerId);
  var hiddenInput = document.getElementById(hiddenInputId);
  if (!container || !hiddenInput) return;

  var initialHTML = hiddenInput.value;

  var quill = new Quill('#' + containerId, {
    theme: 'snow',
    modules: {
      toolbar: [
        [{ header: [1, 2, 3, false] }],
        ['bold', 'italic', 'underline', 'strike'],
        [{ list: 'ordered' }, { list: 'bullet' }],
        ['link', 'blockquote'],
        ['clean']
      ]
    }
  });

  if (initialHTML) {
    quill.clipboard.dangerouslyPasteHTML(initialHTML);
  }

  var form = container.closest('form');
  if (form) {
    form.addEventListener('submit', function () {
      hiddenInput.value = quill.root.innerHTML;
    });
  }
}
