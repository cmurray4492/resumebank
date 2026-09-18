// Wires a file input to POST /candidates/resume/extract and pour the
// resulting HTML into a Quill instance (from initRichTextEditor), so a
// candidate can auto-fill their rich-text resume from an uploaded PDF/DOCX
// instead of typing it. candidateSlug is optional (omitted on signup, where
// no candidate exists yet) and, when present, lets the server also save the
// PDF as the candidate's required resume file if they don't have one yet.
function initResumeExtractUpload(fileInputId, statusId, quill, candidateSlug) {
  var input = document.getElementById(fileInputId);
  var status = document.getElementById(statusId);
  if (!input || !quill) return;

  var form = input.closest('form');
  var csrfInput = form ? form.querySelector('[name="csrf_token"]') : null;

  input.addEventListener('change', function () {
    var file = input.files[0];
    if (!file) return;

    var formData = new FormData();
    formData.append('file', file);
    formData.append('csrf_token', csrfInput ? csrfInput.value : '');
    if (candidateSlug) formData.append('candidate_slug', candidateSlug);

    if (status) {
      status.classList.remove('text-danger');
      status.textContent = 'Reading your resume…';
    }

    fetch('/candidates/resume/extract', { method: 'POST', body: formData })
      .then(function (res) {
        return res.json().then(function (data) {
          return { ok: res.ok, data: data };
        });
      })
      .then(function (result) {
        if (!result.ok) {
          throw new Error(result.data.error || 'Something went wrong reading that file.');
        }
        quill.setText('');
        quill.clipboard.dangerouslyPasteHTML(result.data.html);
        if (status) {
          status.textContent = result.data.resume_pdf_saved
            ? 'Resume text loaded below — review and edit as needed. Your PDF was also saved as your resume file.'
            : 'Resume text loaded below — review and edit as needed.';
        }
      })
      .catch(function (err) {
        if (status) {
          status.classList.add('text-danger');
          status.textContent = err.message;
        }
      })
      .finally(function () {
        input.value = '';
      });
  });
}
