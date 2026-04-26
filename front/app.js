const API_BASE_URL = "http://localhost:8080";
const STORAGE_KEY = "psp-user-front";

const state = {
  accessToken: "",
  refreshToken: "",
  me: null,
  menuOpen: false,
  followingOverrides: {},
  routeSeq: 0,
};

const appEl = document.getElementById("app");
const navEl = document.getElementById("main-nav");
const sessionActionsEl = document.getElementById("session-actions");
const toastEl = document.getElementById("toast");
const menuButton = document.getElementById("mobile-menu-button");
const headerEl = document.querySelector(".site-header");

const navItems = [
  { href: "#/", label: "Главная" },
  { href: "#/feed", label: "Лента" },
  { href: "#/trending", label: "Тренды" },
  { href: "#/tags", label: "Теги" },
  { href: "#/following", label: "Подписки", auth: true },
  { href: "#/create", label: "Создать", auth: true },
  { href: "#/me", label: "Профиль", auth: true },
];

boot();

function boot() {
  hydrateState();
  bindLayout();
  renderChrome();
  ensureRoute();
  window.addEventListener("hashchange", () => {
    state.menuOpen = false;
    renderChrome();
    renderRoute();
  });
  renderRoute();
}

function hydrateState() {
  const raw = localStorage.getItem(STORAGE_KEY);
  if (!raw) {
    return;
  }

  try {
    const parsed = JSON.parse(raw);
    state.accessToken = parsed.accessToken || "";
    state.refreshToken = parsed.refreshToken || "";
    state.me = parsed.me || null;
    state.followingOverrides = parsed.followingOverrides || {};
  } catch {
    clearAuth();
  }
}

function persistState() {
  localStorage.setItem(
    STORAGE_KEY,
    JSON.stringify({
      accessToken: state.accessToken,
      refreshToken: state.refreshToken,
      me: state.me,
      followingOverrides: state.followingOverrides,
    }),
  );
}

function bindLayout() {
  menuButton.addEventListener("click", () => {
    state.menuOpen = !state.menuOpen;
    renderChrome();
  });
}

function renderChrome() {
  const currentHash = location.hash || "#/";
  headerEl.classList.toggle("menu-open", state.menuOpen);

  navEl.innerHTML = navItems
    .filter((item) => !item.auth || state.accessToken)
    .map((item) => {
      const active = item.href === "#/" ? currentHash === "#/" : currentHash.startsWith(item.href);
      return `<a class="${active ? "active" : ""}" href="${item.href}">${escapeHtml(item.label)}</a>`;
    })
    .join("");

  if (state.me) {
    sessionActionsEl.innerHTML = `
      <a class="button secondary" href="#/me">${escapeHtml(state.me.nickname || "Профиль")}</a>
      <button id="logout-button" class="ghost" type="button">Выйти</button>
    `;
    document.getElementById("logout-button").addEventListener("click", logout);
    return;
  }

  sessionActionsEl.innerHTML = `
    <a class="button secondary" href="#/auth">Войти</a>
    <a class="button" href="#/auth?mode=register">Регистрация</a>
  `;
}

function ensureRoute() {
  if (!location.hash) {
    location.hash = "#/";
  }
}

function parseRoute() {
  const raw = location.hash.replace(/^#/, "") || "/";
  const [pathPart, queryPart = ""] = raw.split("?");
  return {
    path: pathPart || "/",
    segments: pathPart.split("/").filter(Boolean),
    query: new URLSearchParams(queryPart),
  };
}

async function renderRoute() {
  const seq = ++state.routeSeq;
  const route = parseRoute();
  appEl.focus({ preventScroll: true });
  appEl.classList.add("is-changing");

  try {
    if (state.accessToken && !state.me) {
      await loadMe(false);
      renderChrome();
    }

    if (route.path === "/") {
      return renderHomePage();
    }
    if (route.path === "/auth") {
      return renderAuthPage(route.query.get("mode") || "login");
    }
    if (route.path === "/feed") {
      return renderFeedPage("Лента", "Новые публичные опросы", "/v1/feed", route.query, { tags: true });
    }
    if (route.path === "/trending") {
      return renderFeedPage("Тренды", "Опросы с активным голосованием", "/v1/feed/trending", route.query);
    }
    if (route.path === "/following") {
      return renderProtected(() => renderFeedPage("Подписки", "Опросы авторов, на которых вы подписаны", "/v1/feed/following", route.query, { auth: true }));
    }
    if (route.path === "/create") {
      return renderProtected(renderCreatePollPage);
    }
    if (route.path === "/me") {
      return renderProtected(renderMePage);
    }
    if (route.path === "/tags") {
      return renderTagsPage();
    }
    if (route.segments[0] === "poll" && route.segments[1]) {
      return renderPollPage(route.segments[1]);
    }
    if (route.segments[0] === "profile" && route.segments[1]) {
      return renderProfilePage(route.segments[1]);
    }

    renderNotFound();
  } catch (error) {
    if (seq !== state.routeSeq) {
      return;
    }
    renderError(error);
  } finally {
    if (seq === state.routeSeq) {
      window.requestAnimationFrame(() => appEl.classList.remove("is-changing"));
    }
  }
}

function renderProtected(renderer) {
  if (!state.accessToken) {
    renderPage(`
      <section class="page-head">
        <div>
          <h1>Нужен вход</h1>
          <p>Авторизуйтесь, чтобы пользоваться этим разделом.</p>
        </div>
        <a class="button" href="#/auth">Войти</a>
      </section>
    `);
    return;
  }

  return renderer();
}

function renderPage(html) {
  appEl.innerHTML = `<div class="page-view">${html}</div>`;
}

function renderLoading(title = "Загрузка") {
  renderPage(`
    <section class="page-head">
      <div>
        <h1>${escapeHtml(title)}</h1>
        <p>Получаем данные...</p>
      </div>
    </section>
    <section class="skeleton-grid">
      <div class="skeleton-card"></div>
      <div class="skeleton-card"></div>
      <div class="skeleton-card"></div>
    </section>
  `);
}

async function renderHomePage() {
  renderPage(`
    <section class="hero">
      <div class="hero-card">
        <h1>Создавайте опросы и смотрите живые ответы.</h1>
        <p>Public Survey помогает публиковать простые вопросы, голосовать, следить за авторами и смотреть аналитику по аудитории.</p>
        <div class="hero-actions">
          ${state.accessToken ? `<a class="button" href="#/create">Создать опрос</a>` : `<a class="button" href="#/auth?mode=register">Начать</a>`}
          <a class="button secondary" href="#/feed">Смотреть ленту</a>
        </div>
      </div>
      <div class="hero-stats" id="home-stats">
        <div class="stat-tile"><strong>...</strong><span>новых опросов</span></div>
        <div class="stat-tile"><strong>...</strong><span>в трендах</span></div>
        <div class="stat-tile"><strong>${state.me ? "1" : "0"}</strong><span>активная сессия</span></div>
      </div>
    </section>

    <section class="grid-three" style="margin-top: 1rem;">
      <article class="card">
        <h3>Голосуйте</h3>
        <p class="muted">Открывайте опрос, выбирайте вариант и сразу смотрите обновлённые результаты.</p>
      </article>
      <article class="card">
        <h3>Подписывайтесь</h3>
        <p class="muted">Следите за авторами и открывайте персональную ленту подписок.</p>
      </article>
      <article class="card">
        <h3>Анализируйте</h3>
        <p class="muted">На странице опроса доступны итоги по вариантам, странам, полу и возрасту.</p>
      </article>
    </section>

    <section id="home-feed" style="margin-top: 1rem;"></section>
  `);

  try {
    const [feed, trending] = await Promise.all([api("/v1/feed?limit=3"), api("/v1/feed/trending?limit=3")]);
    document.getElementById("home-stats").innerHTML = `
      <div class="stat-tile"><strong>${toCount(feed.items?.length || 0)}</strong><span>новых опросов</span></div>
      <div class="stat-tile"><strong>${toCount(trending.items?.length || 0)}</strong><span>в трендах</span></div>
      <div class="stat-tile"><strong>${state.me ? "1" : "0"}</strong><span>активная сессия</span></div>
    `;
    document.getElementById("home-feed").innerHTML = `
      <div class="page-head">
        <div>
          <h1>Свежие опросы</h1>
          <p>Несколько последних публикаций из общей ленты.</p>
        </div>
        <a class="button secondary" href="#/feed">Вся лента</a>
      </div>
      ${renderPollList(feed.items || [])}
    `;
  } catch {
    document.getElementById("home-feed").innerHTML = `<div class="empty">Лента пока недоступна. Проверьте, что backend запущен.</div>`;
  }
}

function renderAuthPage(mode) {
  const isRegister = mode === "register";
  renderPage(`
    <section class="auth-layout">
      <div class="hero-card">
        <h1>${isRegister ? "Создайте аккаунт" : "Вернитесь к опросам"}</h1>
        <p>${isRegister ? "Аккаунт нужен для создания опросов, голосования и подписок." : "Войдите, чтобы голосовать, создавать опросы и видеть свою ленту подписок."}</p>
      </div>
      <section class="card stack">
        <div class="tabbar">
          <button id="login-tab" class="${isRegister ? "" : "active"}" type="button">Вход</button>
          <button id="register-tab" class="${isRegister ? "active" : ""}" type="button">Регистрация</button>
        </div>
        ${isRegister ? renderRegisterForm() : renderLoginForm()}
      </section>
    </section>
  `);

  document.getElementById("login-tab").addEventListener("click", () => {
    location.hash = "#/auth";
  });
  document.getElementById("register-tab").addEventListener("click", () => {
    location.hash = "#/auth?mode=register";
  });
  bindAuthForms();
}

function renderLoginForm() {
  return `
    <form id="login-form" class="stack">
      <label>Email<input name="email" type="email" autocomplete="email" required /></label>
      <label>Пароль<input name="password" type="password" autocomplete="current-password" required /></label>
      <button type="submit">Войти</button>
    </form>
  `;
}

function renderRegisterForm() {
  return `
    <form id="register-form" class="stack">
      <label>Email<input name="email" type="email" autocomplete="email" required /></label>
      <label>Пароль<input name="password" type="password" autocomplete="new-password" minlength="6" required /></label>
      <label>Никнейм<input name="nickname" autocomplete="nickname" required /></label>
      <div class="grid-two">
        <label>Страна<input name="country" placeholder="RU" required /></label>
        <label>Год рождения<input name="birthYear" type="number" min="1900" max="2100" required /></label>
      </div>
      <label>Пол
        <select name="gender" required>
          <option value="male">Мужской</option>
          <option value="female">Женский</option>
          <option value="other">Другой</option>
        </select>
      </label>
      <button type="submit">Создать аккаунт</button>
    </form>
  `;
}

function bindAuthForms() {
  document.getElementById("login-form")?.addEventListener("submit", async (event) => {
    event.preventDefault();
    await submitAuth(event.currentTarget, "/v1/auth/login", false);
  });

  document.getElementById("register-form")?.addEventListener("submit", async (event) => {
    event.preventDefault();
    await submitAuth(event.currentTarget, "/v1/auth/register", true);
  });
}

async function submitAuth(form, path, isRegister) {
  const payload = formToObject(form);
  if (isRegister) {
    payload.birthYear = Number(payload.birthYear);
  }

  const submitButton = form.querySelector("button[type='submit']");
  setButtonBusy(submitButton, isRegister ? "Создаю..." : "Вхожу...");
  try {
    const response = await api(path, { method: "POST", body: payload });
    applyTokens(response.tokens);
    await loadMe(true);
    toast(isRegister ? "Аккаунт создан." : "Вы вошли.");
    renderChrome();
    location.hash = "#/feed";
  } catch (error) {
    toast(error.message, "error");
    setButtonReady(submitButton, isRegister ? "Создать аккаунт" : "Войти");
  }
}

async function renderFeedPage(title, subtitle, path, query, options = {}) {
  renderLoading(title);
  const url = buildListPath(path, query, options);
  const data = await api(url, { auth: !!options.auth });
  const items = data.items || [];
  const routeBase = pathToHash(path);

  renderPage(`
    <section class="page-head">
      <div>
        <h1>${escapeHtml(title)}</h1>
        <p>${escapeHtml(subtitle)}</p>
      </div>
      ${state.accessToken ? `<a class="button" href="#/create">Создать опрос</a>` : `<a class="button" href="#/auth">Войти</a>`}
    </section>
    ${renderFeedControls(query, options)}
    <section id="feed-list-region" class="feed-region">
      ${renderPollList(items)}
    </section>
    ${renderLoadMore(data.page, routeBase, query, path, options)}
  `);

  bindFeedControls(routeBase, options);
  bindLoadMore(path, options);
}

function renderFeedControls(query, options) {
  if (!options.tags) {
    return "";
  }

  return `
    <section class="card" style="margin-bottom: 1rem;">
      <form id="feed-filter-form" class="toolbar">
        <label>Теги<input name="tags" value="${escapeAttr(query.get("tags") || "")}" placeholder="например: sport, news" /></label>
        <label>Показать
          <select name="limit">
            ${renderOption("10", "10", query.get("limit") || "20")}
            ${renderOption("20", "20", query.get("limit") || "20")}
            ${renderOption("50", "50", query.get("limit") || "20")}
          </select>
        </label>
        <button type="submit">Фильтровать</button>
        <a class="button secondary" href="#/feed">Сбросить</a>
      </form>
    </section>
  `;
}

function bindFeedControls(routeBase, options) {
  document.getElementById("feed-filter-form")?.addEventListener("submit", (event) => {
    event.preventDefault();
    const payload = formToObject(event.currentTarget);
    location.hash = withQuery(routeBase, {
      tags: options.tags ? payload.tags : "",
      limit: payload.limit || "20",
    });
  });
}

function renderPollList(items) {
  if (!items.length) {
    return `<div class="empty">Пока здесь нет опросов.</div>`;
  }

  return `<div class="feed-list">${items.map(renderPollCard).join("")}</div>`;
}

function renderPollCard(item) {
  const author = item.author || {};
  const options = item.options || [];
  const totalVotes = Number(item.totalVotes || 0);
  const leader = options.reduce((best, option) => (Number(option.votesCount || 0) > Number(best?.votesCount || 0) ? option : best), options[0] || null);

  return `
    <article class="card poll-card">
      <div class="poll-card-header">
        <div>
          <h3><a href="#/poll/${encodeURIComponent(item.id)}">${escapeHtml(item.question || "Без вопроса")}</a></h3>
          <div class="small muted">
            ${author.id || item.creatorId ? `Автор: <a href="#/profile/${encodeURIComponent(author.id || item.creatorId)}">${escapeHtml(author.nickname || author.id || item.creatorId)}</a>` : "Автор неизвестен"}
          </div>
        </div>
        <span class="chip">${formatDate(item.createdAt)}</span>
      </div>
      ${item.imageUrl ? `<img class="poll-image" src="${escapeAttr(item.imageUrl)}" alt="" loading="lazy" />` : ""}
      <div class="option-list">
        ${options.slice(0, 4).map((option) => renderResultRow(option.text, option.votesCount, totalVotes)).join("")}
      </div>
      <div class="footer-actions">
        <span class="tag">${toCount(totalVotes)} голосов</span>
        ${leader ? `<span class="chip">Лидер: ${escapeHtml(leader.text || "-")}</span>` : ""}
        ${(item.tags || []).map((tag) => `<a class="tag" href="#/feed?tags=${encodeURIComponent(tag)}">${escapeHtml(tag)}</a>`).join("")}
        <a class="button secondary" href="#/poll/${encodeURIComponent(item.id)}">Открыть</a>
      </div>
    </article>
  `;
}

function renderResultRow(label, votes, totalVotes) {
  const percent = totalVotes > 0 ? Math.round((Number(votes || 0) / totalVotes) * 100) : 0;
  return `
    <div class="option-row">
      <span>${escapeHtml(label || "-")}</span>
      <div class="bar" aria-hidden="true"><span style="--value: ${percent}%"></span></div>
      <strong>${toCount(votes)}</strong>
    </div>
  `;
}

async function renderCreatePollPage() {
  renderLoading("Новый опрос");
  const tags = await safeLoadTags();

  renderPage(`
    <section class="page-head">
      <div>
        <h1>Создать опрос</h1>
        <p>Добавьте вопрос, варианты ответа и при необходимости изображение.</p>
      </div>
    </section>
    <div class="split">
      <section class="card">
        <form id="create-poll-form" class="stack">
          <label>Вопрос<textarea name="question" maxlength="300" required placeholder="О чём спросим аудиторию?"></textarea></label>
          <label>Тип голосования
            <select name="type">
              <option value="POLL_TYPE_SINGLE_CHOICE">Один вариант</option>
              <option value="POLL_TYPE_MULTIPLE_CHOICE">Несколько вариантов</option>
            </select>
          </label>
          <label>Варианты ответа<textarea name="options" required placeholder="Да&#10;Нет&#10;Пока не знаю"></textarea></label>
          <label>Теги<input name="tags" placeholder="news, sport, city" /></label>
          <label>Изображение<input id="poll-image-file" type="file" accept="image/*" /></label>
          <button type="submit">Опубликовать</button>
        </form>
      </section>
      <aside class="card stack sticky-panel">
        <div>
          <h3>Предпросмотр</h3>
          <div id="poll-preview" class="poll-preview"></div>
        </div>
        <h3>Популярные теги</h3>
        <div class="tag-list">${tags.map((tag) => `<button class="secondary tag-pick" type="button" data-tag="${escapeAttr(tag.name)}">${escapeHtml(tag.name)}</button>`).join("") || `<span class="muted">Тегов пока нет.</span>`}</div>
        <p class="hint small">Теги помогают пользователям находить опросы в ленте.</p>
        <a class="button secondary" href="#/tags">Управлять тегами</a>
      </aside>
    </div>
  `);

  const form = document.getElementById("create-poll-form");
  document.querySelectorAll(".tag-pick").forEach((button) => {
    button.addEventListener("click", () => {
      appendTag(form.elements.tags, button.dataset.tag);
      updatePollPreview(form);
    });
  });
  form.addEventListener("input", () => updatePollPreview(form));
  updatePollPreview(form);
  form.addEventListener("submit", handleCreatePoll);
}

function updatePollPreview(form) {
  const preview = document.getElementById("poll-preview");
  if (!preview) {
    return;
  }
  const payload = formToObject(form);
  const options = String(payload.options || "")
    .split("\n")
    .map((item) => item.trim())
    .filter(Boolean);
  const tags = splitCSV(payload.tags);

  preview.innerHTML = `
    <article class="mini-poll">
      <h4>${escapeHtml(payload.question || "Ваш вопрос появится здесь")}</h4>
      <div class="option-list">
        ${(options.length ? options : ["Первый вариант", "Второй вариант"])
          .slice(0, 4)
          .map((option) => `<div class="option-row"><span>${escapeHtml(option)}</span><div class="bar"><span style="--value: 0%"></span></div><strong>0</strong></div>`)
          .join("")}
      </div>
      <div class="tag-list">${tags.map((tag) => `<span class="tag">${escapeHtml(tag)}</span>`).join("") || `<span class="chip">без тегов</span>`}</div>
    </article>
  `;
}

async function handleCreatePoll(event) {
  event.preventDefault();
  const form = event.currentTarget;
  const payload = formToObject(form);
  const options = String(payload.options || "")
    .split("\n")
    .map((item) => item.trim())
    .filter(Boolean);

  if (options.length < 2) {
    toast("Добавьте минимум два варианта ответа.", "error");
    return;
  }

  try {
    const submitButton = form.querySelector("button[type='submit']");
    setButtonBusy(submitButton, "Публикую...");
    const file = document.getElementById("poll-image-file").files[0];
    const imageUrl = file ? await uploadImage(file) : "";
    const response = await api("/v1/polls", {
      method: "POST",
      auth: true,
      body: {
        question: payload.question,
        type: payload.type,
        options,
        tags: splitCSV(payload.tags),
        imageUrl,
      },
    });

    toast("Опрос опубликован.");
    location.hash = `#/poll/${encodeURIComponent(response.poll.id)}`;
  } catch (error) {
    toast(error.message, "error");
    setButtonReady(form.querySelector("button[type='submit']"), "Опубликовать");
  }
}

async function renderPollPage(pollId) {
  renderLoading("Опрос");
  const pollResponse = await api(`/v1/polls/${encodeURIComponent(pollId)}`);
  const poll = pollResponse.poll;

  const requests = [
    api(`/v1/profiles/${encodeURIComponent(poll.creatorId)}`, { auth: !!state.accessToken }),
    api(`/v1/polls/${encodeURIComponent(pollId)}/analytics`),
    api(`/v1/polls/${encodeURIComponent(pollId)}/analytics/countries`),
    api(`/v1/polls/${encodeURIComponent(pollId)}/analytics/gender`),
    api(`/v1/polls/${encodeURIComponent(pollId)}/analytics/age`),
  ];
  if (state.accessToken) {
    requests.push(api(`/v1/polls/${encodeURIComponent(pollId)}/vote`, { auth: true }));
  }

  const results = await Promise.allSettled(requests);
  const author = settledValue(results[0], { profile: { id: poll.creatorId, nickname: "Автор" } }).profile;
  const analytics = settledValue(results[1], { totalVotes: poll.totalVotes, options: [] });
  const countries = settledValue(results[2], { items: [] });
  const gender = settledValue(results[3], { items: [] });
  const age = settledValue(results[4], { items: [] });
  const vote = settledValue(results[5], { hasVoted: false, optionIds: [] });
  const isOwner = state.me && state.me.id === poll.creatorId;
  const isMultiple = poll.type === "POLL_TYPE_MULTIPLE_CHOICE";

  renderPage(`
    <section class="page-head">
      <div>
        <h1>${escapeHtml(poll.question)}</h1>
        <p>Автор: <a href="#/profile/${encodeURIComponent(poll.creatorId)}">${escapeHtml(author.nickname || poll.creatorId)}</a></p>
      </div>
      <span class="chip">${formatDate(poll.createdAt)}</span>
    </section>

    <div class="split">
      <section class="card stack">
        ${poll.imageUrl ? `<img class="poll-image" src="${escapeAttr(poll.imageUrl)}" alt="" />` : ""}
        <div class="tag-list">${(poll.tags || []).map((tag) => `<a class="tag" href="#/feed?tags=${encodeURIComponent(tag)}">${escapeHtml(tag)}</a>`).join("")}</div>
        <div class="metric-row">
          <div class="metric"><strong>${toCount(poll.totalVotes)}</strong><span>голосов</span></div>
          <div class="metric"><strong>${poll.options?.length || 0}</strong><span>вариантов</span></div>
          <div class="metric"><strong>${isMultiple ? "multi" : "single"}</strong><span>тип</span></div>
        </div>

        ${state.accessToken ? renderVoteForm(poll, vote, isMultiple) : `<div class="empty">Войдите, чтобы проголосовать. <a href="#/auth">Авторизация</a></div>`}
        ${isOwner ? renderOwnerControls(poll) : ""}
      </section>

      <aside class="card stack">
        <div class="tabbar compact-tabs">
          <button class="active" type="button" data-tab="results">Результаты</button>
          <button type="button" data-tab="analytics">Аналитика</button>
        </div>
        <section data-tab-panel="results">
          <h3>Результаты</h3>
          <div class="option-list">
            ${(poll.options || []).map((option) => renderResultRow(option.text, option.votesCount, poll.totalVotes)).join("")}
          </div>
        </section>
        <section class="hidden" data-tab-panel="analytics">
          <h3>Аналитика</h3>
          ${renderAnalytics(poll, analytics, countries, gender, age)}
        </section>
      </aside>
    </div>
  `);

  bindPollActions(poll, isMultiple, vote);
  bindLocalTabs();
}

function renderVoteForm(poll, vote, isMultiple) {
  const inputType = isMultiple ? "checkbox" : "radio";
  const selected = new Set(vote.optionIds || []);

  return `
    <form id="vote-form" class="stack">
      <h3>${vote.hasVoted ? "Ваш голос" : "Ваш выбор"}</h3>
      <div class="option-list">
        ${(poll.options || [])
          .map(
            (option) => `
              <label class="vote-option">
                <input type="${inputType}" name="optionId" value="${escapeAttr(option.id)}" ${selected.has(option.id) ? "checked" : ""} />
                <span>${escapeHtml(option.text)}</span>
                <strong>${toCount(option.votesCount)}</strong>
              </label>
            `,
          )
          .join("")}
      </div>
      <div class="actions">
        <button type="submit">${vote.hasVoted ? "Изменить голос" : "Проголосовать"}</button>
        <button id="remove-vote-button" class="secondary" type="button" ${vote.hasVoted ? "" : "disabled"}>Убрать голос</button>
      </div>
    </form>
  `;
}

function renderOwnerControls(poll) {
  return `
    <section class="card stack">
      <h3>Управление опросом</h3>
      <form id="edit-poll-form" class="stack">
        <label>Вопрос<textarea name="question" required>${escapeHtml(poll.question)}</textarea></label>
        <label>Теги<input name="tags" value="${escapeAttr((poll.tags || []).join(", "))}" /></label>
        <label>Заменить изображение<input id="edit-poll-image-file" type="file" accept="image/*" /></label>
        <div class="actions">
          <button type="submit">Сохранить</button>
          <button id="delete-poll-button" class="danger" type="button">Удалить</button>
        </div>
      </form>
    </section>
  `;
}

function bindPollActions(poll, isMultiple) {
  document.getElementById("vote-form")?.addEventListener("submit", async (event) => {
    event.preventDefault();
    const optionIds = [...event.currentTarget.querySelectorAll("input[name='optionId']:checked")].map((input) => input.value);
    if (!optionIds.length) {
      toast("Выберите вариант.", "error");
      return;
    }
    if (!isMultiple && optionIds.length > 1) {
      toast("В этом опросе можно выбрать только один вариант.", "error");
      return;
    }

    const submitButton = event.currentTarget.querySelector("button[type='submit']");
    setButtonBusy(submitButton, "Сохраняю...");
    try {
      await api(`/v1/polls/${encodeURIComponent(poll.id)}/vote`, {
        method: "POST",
        auth: true,
        body: { optionIds },
      });
      toast("Голос сохранён.");
      await renderPollPage(poll.id);
    } catch (error) {
      toast(error.message, "error");
      setButtonReady(submitButton, "Проголосовать");
    }
  });

  document.getElementById("remove-vote-button")?.addEventListener("click", async () => {
    const button = document.getElementById("remove-vote-button");
    setButtonBusy(button, "Удаляю...");
    try {
      await api(`/v1/polls/${encodeURIComponent(poll.id)}/vote`, { method: "DELETE", auth: true });
      toast("Голос удалён.");
      await renderPollPage(poll.id);
    } catch (error) {
      toast(error.message, "error");
      setButtonReady(button, "Убрать голос");
    }
  });

  document.getElementById("edit-poll-form")?.addEventListener("submit", async (event) => {
    event.preventDefault();
    const payload = formToObject(event.currentTarget);
    payload.tags = splitCSV(payload.tags);

    const submitButton = event.currentTarget.querySelector("button[type='submit']");
    setButtonBusy(submitButton, "Сохраняю...");
    try {
      const file = document.getElementById("edit-poll-image-file").files[0];
      if (file) {
        payload.imageUrl = await uploadImage(file);
      }
      await api(`/v1/polls/${encodeURIComponent(poll.id)}`, { method: "PATCH", auth: true, body: payload });
      toast("Опрос обновлён.");
      await renderPollPage(poll.id);
    } catch (error) {
      toast(error.message, "error");
      setButtonReady(submitButton, "Сохранить");
    }
  });

  document.getElementById("delete-poll-button")?.addEventListener("click", async () => {
    if (!window.confirm("Удалить опрос без восстановления?")) {
      return;
    }
    try {
      await api(`/v1/polls/${encodeURIComponent(poll.id)}`, { method: "DELETE", auth: true });
      toast("Опрос удалён.");
      location.hash = "#/me";
    } catch (error) {
      toast(error.message, "error");
    }
  });
}

function renderAnalytics(poll, analytics, countries, gender, age) {
  const optionNames = new Map((poll.options || []).map((option) => [option.id, option.text]));
  return `
    <div class="stats-list">
      <div class="stats-item"><span>Всего</span><strong>${toCount(analytics.totalVotes)}</strong></div>
      ${(analytics.options || [])
        .map((item) => `<div class="stats-item"><span>${escapeHtml(optionNames.get(item.optionId) || item.optionId)}</span><strong>${toCount(item.votes)}</strong></div>`)
        .join("")}
    </div>
    <h4>Страны</h4>
    ${renderStats(countries.items, "country")}
    <h4>Пол</h4>
    ${renderStats(gender.items, "gender")}
    <h4>Возраст</h4>
    ${renderStats(age.items, "ageRange")}
  `;
}

function renderStats(items = [], key) {
  if (!items.length) {
    return `<div class="empty">Данных пока нет.</div>`;
  }
  return `<div class="stats-list">${items.map((item) => `<div class="stats-item"><span>${escapeHtml(item[key] || "-")}</span><strong>${toCount(item.votes)}</strong></div>`).join("")}</div>`;
}

async function renderProfilePage(userId) {
  renderLoading("Профиль");
  const [profileResponse, pollsResponse] = await Promise.all([
    api(`/v1/profiles/${encodeURIComponent(userId)}`, { auth: !!state.accessToken }),
    api(`/v1/users/${encodeURIComponent(userId)}/polls?limit=20`),
  ]);
  const profile = profileResponse.profile;
  if (Object.prototype.hasOwnProperty.call(state.followingOverrides, profile.id)) {
    profile.isFollowing = state.followingOverrides[profile.id];
  }
  const isOwn = state.me && state.me.id === profile.id;

  renderPage(`
    <section class="page-head">
      <div>
        <h1>${escapeHtml(profile.nickname || "Пользователь")}</h1>
        <p>Публичный профиль и опубликованные опросы.</p>
      </div>
      ${renderFollowButton(profile, isOwn)}
    </section>
    <div class="split">
      <aside class="card profile-card">
        <div class="profile-avatar">${escapeHtml(initials(profile.nickname || profile.id))}</div>
        <h2>${escapeHtml(profile.nickname || "-")}</h2>
        <div class="metric-row">
          <div class="metric"><strong>${toCount(profile.followersCount)}</strong><span>подписчиков</span></div>
          <div class="metric"><strong>${toCount(profile.followingCount)}</strong><span>подписок</span></div>
          <div class="metric"><strong>${toCount(pollsResponse.items?.length || 0)}</strong><span>опросов</span></div>
        </div>
      </aside>
      <section class="stack">
        ${renderPollList(pollsResponse.items || [])}
      </section>
    </div>
  `);

  document.getElementById("follow-button")?.addEventListener("click", async () => {
    try {
      await api(`/v1/users/${encodeURIComponent(profile.id)}:follow`, {
        method: profile.isFollowing ? "DELETE" : "POST",
        auth: true,
        body: profile.isFollowing ? undefined : {},
      });
      state.followingOverrides[profile.id] = !profile.isFollowing;
      persistState();
      toast(profile.isFollowing ? "Вы отписались." : "Вы подписались.");
      await renderProfilePage(profile.id);
    } catch (error) {
      toast(error.message, "error");
    }
  });
}

function renderFollowButton(profile, isOwn) {
  if (!state.accessToken) {
    return `<a class="button" href="#/auth">Войти, чтобы подписаться</a>`;
  }
  if (isOwn) {
    return `<span class="chip">Это вы</span>`;
  }
  return `<button id="follow-button" type="button">${profile.isFollowing ? "Отписаться" : "Подписаться"}</button>`;
}

async function renderMePage() {
  renderLoading("Мой профиль");
  const meResponse = await api("/v1/users/me", { auth: true });
  state.me = meResponse.user;
  persistState();
  renderChrome();

  const myPolls = await api("/v1/feed/me?limit=20", { auth: true }).catch(() => ({ items: [] }));

  renderPage(`
    <section class="page-head">
      <div>
        <h1>Мой профиль</h1>
        <p>Личные данные и ваши опубликованные опросы.</p>
      </div>
      <a class="button" href="#/create">Создать опрос</a>
    </section>
    <div class="split">
      <section class="card stack">
        <div class="profile-avatar">${escapeHtml(initials(state.me.nickname || state.me.email))}</div>
        <h2>${escapeHtml(state.me.nickname || "-")}</h2>
        <form id="profile-form" class="stack">
          <label>Email<input name="email" type="email" value="${escapeAttr(state.me.email || "")}" /></label>
          <label>Никнейм<input name="nickname" value="${escapeAttr(state.me.nickname || "")}" /></label>
          <div class="grid-two">
            <label>Страна<input name="country" value="${escapeAttr(state.me.country || "")}" /></label>
            <label>Год рождения<input name="birthYear" type="number" value="${escapeAttr(state.me.birthYear || "")}" /></label>
          </div>
          <label>Пол<input name="gender" value="${escapeAttr(state.me.gender || "")}" /></label>
          <button type="submit">Сохранить профиль</button>
        </form>
      </section>
      <section class="stack">
        <div class="card">
          <h2>Мои опросы</h2>
          <p class="muted">Здесь отображается лента ваших публикаций.</p>
        </div>
        ${renderPollList(myPolls.items || [])}
      </section>
    </div>
  `);

  document.getElementById("profile-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    const payload = formToObject(event.currentTarget);
    if (payload.birthYear) {
      payload.birthYear = Number(payload.birthYear);
    } else {
      delete payload.birthYear;
    }

    try {
      const response = await api("/v1/users/me", { method: "PATCH", auth: true, body: payload });
      state.me = response.user;
      persistState();
      renderChrome();
      toast("Профиль обновлён.");
      await renderMePage();
    } catch (error) {
      toast(error.message, "error");
    }
  });
}

async function renderTagsPage() {
  renderLoading("Теги");
  const tags = await safeLoadTags();
  renderPage(`
    <section class="page-head">
      <div>
        <h1>Теги</h1>
        <p>Используйте теги, чтобы группировать опросы.</p>
      </div>
    </section>
    <div class="grid-two">
      <section class="card stack">
        <h2>Все теги</h2>
        <div class="tag-list">${tags.map((tag) => `<a class="tag" href="#/feed?tags=${encodeURIComponent(tag.name)}">${escapeHtml(tag.name)}</a>`).join("") || `<span class="muted">Тегов пока нет.</span>`}</div>
      </section>
      <section class="card stack">
        <h2>Новый тег</h2>
        ${state.accessToken ? `
          <form id="tag-form" class="stack">
            <label>Название<input name="name" required /></label>
            <button type="submit">Создать тег</button>
          </form>
        ` : `<div class="empty">Войдите, чтобы создавать теги.</div>`}
      </section>
    </div>
  `);

  document.getElementById("tag-form")?.addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      await api("/v1/tags", { method: "POST", auth: true, body: formToObject(event.currentTarget) });
      toast("Тег создан.");
      await renderTagsPage();
    } catch (error) {
      toast(error.message, "error");
    }
  });
}

function renderLoadMore(page, routeBase, query, path, options) {
  if (!page?.hasMore || !page.nextCursor) {
    return "";
  }

  return `
    <div class="actions load-more-row" style="margin-top: 1rem;">
      <button
        id="load-more-button"
        class="secondary"
        type="button"
        data-path="${escapeAttr(path)}"
        data-route="${escapeAttr(routeBase)}"
        data-limit="${escapeAttr(page.limit || query.get("limit") || "20")}"
        data-cursor="${escapeAttr(page.nextCursor)}"
        data-tags="${escapeAttr(query.get("tags") || "")}"
        data-auth="${options.auth ? "true" : "false"}"
        data-tags-enabled="${options.tags ? "true" : "false"}"
      >Показать ещё</button>
    </div>
  `;
}

function bindLoadMore(defaultPath, defaultOptions = {}) {
  document.getElementById("load-more-button")?.addEventListener("click", async (event) => {
    const button = event.currentTarget;
    const path = button.dataset.path || defaultPath;
    const options = {
      auth: button.dataset.auth === "true" || !!defaultOptions.auth,
      tags: button.dataset.tagsEnabled === "true" || !!defaultOptions.tags,
    };
    const query = new URLSearchParams();
    query.set("limit", button.dataset.limit || "20");
    query.set("cursor", button.dataset.cursor || "");
    if (button.dataset.tags) {
      query.set("tags", button.dataset.tags);
    }

    setButtonBusy(button, "Загружаю...");
    try {
      const data = await api(buildListPath(path, query, options), { auth: !!options.auth });
      appendPollItems(data.items || []);
      const next = renderLoadMore(data.page, button.dataset.route || pathToHash(path), query, path, options);
      document.querySelector(".load-more-row")?.remove();
      if (next) {
        document.getElementById("feed-list-region").insertAdjacentHTML("afterend", next);
        bindLoadMore(path, options);
      }
    } catch (error) {
      toast(error.message, "error");
      setButtonReady(button, "Показать ещё");
    }
  });
}

function appendPollItems(items) {
  const region = document.getElementById("feed-list-region");
  if (!region || !items.length) {
    return;
  }

  let list = region.querySelector(".feed-list");
  if (!list) {
    region.innerHTML = `<div class="feed-list"></div>`;
    list = region.querySelector(".feed-list");
  }
  list.insertAdjacentHTML("beforeend", items.map(renderPollCard).join(""));
}

async function safeLoadTags() {
  try {
    const response = await api("/v1/tags");
    return response.items || [];
  } catch {
    return [];
  }
}

async function uploadImage(file) {
  const upload = await api("/v1/polls/images:upload-url", {
    method: "POST",
    auth: true,
    body: {
      filename: file.name,
      contentType: file.type || "application/octet-stream",
      sizeBytes: file.size,
    },
  });

  const response = await fetch(`/__upload_proxy?uploadUrl=${encodeURIComponent(upload.uploadUrl)}`, {
    method: "POST",
    headers: {
      "X-Upload-Content-Type": file.type || "application/octet-stream",
    },
    body: file,
  });

  if (!response.ok) {
    const text = await response.text().catch(() => "");
    throw new Error(`Не удалось загрузить изображение: ${text || response.status}`);
  }

  return upload.imageUrl;
}

async function logout() {
  try {
    if (state.refreshToken) {
      await api("/v1/auth/logout", {
        method: "POST",
        auth: true,
        body: { refreshToken: state.refreshToken },
      });
    }
  } catch {
    // Local logout still has to happen if server-side revoke failed.
  }

  clearAuth();
  renderChrome();
  toast("Вы вышли.");
  location.hash = "#/";
}

function clearAuth() {
  state.accessToken = "";
  state.refreshToken = "";
  state.me = null;
  state.followingOverrides = {};
  persistState();
}

function applyTokens(tokens) {
  state.accessToken = tokens?.accessToken || "";
  state.refreshToken = tokens?.refreshToken || state.refreshToken || "";
  persistState();
}

async function loadMe(force = false) {
  if (!state.accessToken) {
    return null;
  }
  if (state.me && !force) {
    return state.me;
  }

  try {
    const response = await api("/v1/users/me", { auth: true });
    state.me = response.user;
    persistState();
    return state.me;
  } catch (error) {
    clearAuth();
    if (force) {
      throw error;
    }
    return null;
  }
}

async function refreshTokens() {
  if (!state.refreshToken) {
    throw new Error("Сессия истекла. Войдите снова.");
  }

  const response = await api("/v1/auth/refresh", {
    method: "POST",
    body: { refreshToken: state.refreshToken },
    skipRefresh: true,
  });
  applyTokens(response.tokens);
  return response.tokens;
}

async function api(path, options = {}) {
  const { method = "GET", body, auth = false, skipRefresh = false } = options;
  const headers = { Accept: "application/json" };
  if (body !== undefined) {
    headers["Content-Type"] = "application/json";
  }
  if (auth && state.accessToken) {
    headers.Authorization = `Bearer ${state.accessToken}`;
  }

  const response = await fetch(API_BASE_URL + path, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });
  const payload = await parseResponse(response);

  if (response.status === 401 && auth && state.refreshToken && !skipRefresh) {
    try {
      await refreshTokens();
      return api(path, { ...options, skipRefresh: true });
    } catch {
      clearAuth();
      renderChrome();
    }
  }

  if (!response.ok) {
    throw new Error(readableError(payload, response.status));
  }

  return payload;
}

async function parseResponse(response) {
  const text = await response.text();
  if (!text) {
    return {};
  }
  try {
    return JSON.parse(text);
  } catch {
    return { message: text };
  }
}

function readableError(payload, status) {
  const raw = payload?.message || payload?.error || `Ошибка ${status}`;
  const clean = String(raw).replace(/^rpc error: code = \w+ desc = /, "");
  const messages = {
    "missing or invalid bearer token": "Войдите в аккаунт.",
    "invalid token": "Сессия истекла. Войдите снова.",
    "email already exists": "Email уже занят.",
    "nickname already exists": "Никнейм уже занят.",
    "cannot follow self": "Нельзя подписаться на себя.",
    "not found": "Запись не найдена.",
  };
  return messages[clean] || clean;
}

function buildListPath(path, query, options = {}) {
  const params = new URLSearchParams();
  params.set("limit", query.get("limit") || "20");
  if (query.get("cursor")) {
    params.set("cursor", query.get("cursor"));
  }
  if (options.tags && query.get("tags")) {
    splitCSV(query.get("tags")).forEach((tag) => params.append("tags", tag));
  }
  return `${path}?${params.toString()}`;
}

function pathToHash(path) {
  if (path === "/v1/feed/trending") {
    return "#/trending";
  }
  if (path === "/v1/feed/following") {
    return "#/following";
  }
  return "#/feed";
}

function withQuery(routeBase, values) {
  const params = new URLSearchParams();
  Object.entries(values).forEach(([key, value]) => {
    const normalized = String(value ?? "").trim();
    if (normalized) {
      params.set(key, normalized);
    }
  });
  const query = params.toString();
  return query ? `${routeBase}?${query}` : routeBase;
}

function renderOption(value, label, selected) {
  return `<option value="${escapeAttr(value)}" ${value === selected ? "selected" : ""}>${escapeHtml(label)}</option>`;
}

function appendTag(input, tag) {
  const current = splitCSV(input.value);
  if (!current.includes(tag)) {
    current.push(tag);
  }
  input.value = current.join(", ");
}

function bindLocalTabs() {
  document.querySelectorAll("[data-tab]").forEach((button) => {
    button.addEventListener("click", () => {
      const tab = button.dataset.tab;
      const container = button.closest(".card") || document;
      container.querySelectorAll("[data-tab]").forEach((item) => item.classList.toggle("active", item === button));
      container.querySelectorAll("[data-tab-panel]").forEach((panel) => {
        panel.classList.toggle("hidden", panel.dataset.tabPanel !== tab);
      });
    });
  });
}

function setButtonBusy(button, label = "Загрузка...") {
  if (!button) {
    return;
  }
  button.dataset.readyLabel = button.dataset.readyLabel || button.textContent;
  button.textContent = label;
  button.disabled = true;
  button.classList.add("is-busy");
}

function setButtonReady(button, label = "") {
  if (!button) {
    return;
  }
  button.textContent = label || button.dataset.readyLabel || button.textContent;
  button.disabled = false;
  button.classList.remove("is-busy");
}

function formToObject(form) {
  const data = new FormData(form);
  const out = {};
  for (const [key, value] of data.entries()) {
    if (value instanceof File) {
      continue;
    }
    out[key] = String(value).trim();
  }
  return out;
}

function splitCSV(value) {
  return String(value || "")
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

function settledValue(result, fallback) {
  return result?.status === "fulfilled" ? result.value : fallback;
}

function initials(value) {
  return String(value || "?").trim().slice(0, 2).toUpperCase();
}

function formatDate(value) {
  if (!value) {
    return "";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return String(value);
  }
  return date.toLocaleDateString("ru-RU", { day: "2-digit", month: "short", year: "numeric" });
}

function toCount(value) {
  const number = Number(value || 0);
  return Number.isNaN(number) ? String(value || 0) : number.toLocaleString("ru-RU");
}

function toast(message, type = "info") {
  toastEl.textContent = message;
  toastEl.className = `toast ${type === "error" ? "error" : ""}`;
  window.clearTimeout(toast._timer);
  toast._timer = window.setTimeout(() => {
    toastEl.className = "toast hidden";
  }, 4200);
}

function renderNotFound() {
  renderPage(`
    <section class="page-head">
      <div>
        <h1>Страница не найдена</h1>
        <p>Такого раздела нет.</p>
      </div>
      <a class="button" href="#/">На главную</a>
    </section>
  `);
}

function renderError(error) {
  renderPage(`
    <section class="page-head">
      <div>
        <h1>Что-то пошло не так</h1>
        <p>${escapeHtml(error.message || "Не удалось загрузить данные.")}</p>
      </div>
      <button id="retry-route-button" type="button">Повторить</button>
    </section>
  `);
  document.getElementById("retry-route-button")?.addEventListener("click", renderRoute);
}

function escapeHtml(value) {
  return String(value ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

function escapeAttr(value) {
  return escapeHtml(value);
}
