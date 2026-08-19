document.getElementById('register-form').addEventListener('submit', async (e) => {
  e.preventDefault();

  const username = document.getElementById('username').value.trim();
  const password = document.getElementById('password').value;
  const languageName = document.getElementById('language-name').value.trim();
  const errorEl = document.getElementById('error');
  errorEl.classList.add('hidden');

  const body = { username, password };
  if (languageName) body.languageName = languageName;

  try {
    await apiFetch('/auth/register', {
      method: 'POST',
      body: JSON.stringify(body),
    });

    // Registration doesn't establish a browser session by itself
    // (it only returns an API key, meant for MCP/programmatic use) —
    // log in immediately after so the user lands in the app signed in.
    await apiFetch('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    });

    window.location.href = '/app.html';
  } catch (err) {
    errorEl.textContent = 'Could not create account. Username may already be taken.';
    errorEl.classList.remove('hidden');
  }
});
