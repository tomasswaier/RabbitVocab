document.getElementById('login-form').addEventListener('submit', async (e) => {
  e.preventDefault();

  const username = document.getElementById('username').value.trim();
  const password = document.getElementById('password').value;
  const errorEl = document.getElementById('error');
  errorEl.classList.add('hidden');

  try {
    await apiFetch('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    });
    window.location.href = '/app.html';
  } catch (err) {
    errorEl.textContent = 'Invalid username or password';
    errorEl.classList.remove('hidden');
  }
});
