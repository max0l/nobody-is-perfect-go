const state = {
  userUUID: localStorage.getItem("nip.userUUID") || "",
  username: localStorage.getItem("nip.username") || "",
  gameId: document.body.dataset.gameId || localStorage.getItem("nip.gameId") || "",
  status: null,
  answers: [],
  revealed: [],
  votedAnswerUUID: "",
  systemStatus: null,
  pollTimer: 0,
  busy: false,
  menuOpen: false,
};

const stageCard = document.querySelector("#stage-card");
const message = document.querySelector("#message");

function init() {
  bindGlobalEvents();
  renderStage();
  if (state.gameId && state.userUUID) {
    enterGame(state.gameId, false);
  }
}

function bindGlobalEvents() {
  window.addEventListener("popstate", () => {
    const match = location.pathname.match(/^\/game\/([^/]+)$/);
    if (match) {
      enterGame(decodeURIComponent(match[1]), true);
      return;
    }
    state.gameId = "";
    state.status = null;
    state.menuOpen = false;
    stopTimers();
    renderStage();
  });
}

function bindStageEvents() {
  const profileForm = document.querySelector("#profile-form");
  if (profileForm) {
    profileForm.addEventListener("submit", async (event) => {
      event.preventDefault();
      const username = fieldValue("username");
      if (!username) return;
      await withBusy(async () => {
        const user = await api("/api/create/user", { method: "POST", body: { username } });
        state.userUUID = user.userUUID || "";
        state.username = username;
        localStorage.setItem("nip.userUUID", state.userUUID);
        localStorage.setItem("nip.username", username);
        notice(`Session started for ${username}`);
        if (state.gameId) {
          renderStage("choose");
        } else {
          renderStage();
        }
      });
    });
  }

  const createGame = document.querySelector("#create-game");
  if (createGame) {
    createGame.addEventListener("click", async () => {
      await withBusy(async () => {
        const game = await api("/api/create/game", { method: "POST" });
        await enterGame(game.gameId, false);
        notice(`Created game ${game.gameId}`);
      });
    });
  }

  const joinGameToggle = document.querySelector("#join-game-toggle");
  if (joinGameToggle) {
    joinGameToggle.addEventListener("click", () => {
      const joinForm = document.querySelector("#join-form");
      if (!joinForm) return;
      const isHidden = joinForm.hidden;
      joinForm.hidden = !isHidden;
      joinGameToggle.setAttribute("aria-expanded", String(isHidden));
      if (isHidden) document.querySelector("#join-game-id")?.focus();
    });
  }

  const joinForm = document.querySelector("#join-form");
  if (joinForm) {
    joinForm.addEventListener("submit", async (event) => {
      event.preventDefault();
      const gameId = fieldValue("join-game-id");
      if (!gameId) return;
      await withBusy(async () => {
        await api(`/api/join/${encodeURIComponent(gameId)}`, { method: "POST" });
        await enterGame(gameId, false);
        notice(`Joined game ${gameId}`);
      });
    });
  }

  const copyLink = document.querySelector("#copy-link");
  if (copyLink) {
    copyLink.addEventListener("click", async () => {
      await navigator.clipboard.writeText(inviteLink());
      notice("Invite link copied");
    });
  }

  const shareLink = document.querySelector("#share-link");
  if (shareLink) {
    shareLink.addEventListener("click", async () => {
      const url = inviteLink();
      try {
        if (navigator.share) {
          await navigator.share({ title: "Join my Nobody is Perfect game", text: `Join game ${state.gameId}`, url });
          return;
        }
        await navigator.clipboard.writeText(url);
        notice("Invite link copied");
      } catch (error) {
        if (error.name !== "AbortError") notice(error.message);
      }
    });
  }

  const menuToggle = document.querySelector("#menu-toggle");
  if (menuToggle) {
    menuToggle.addEventListener("click", () => {
      state.menuOpen = !state.menuOpen;
      renderStage();
    });
  }

  const systemStatus = document.querySelector("#system-status");
  if (systemStatus) {
    systemStatus.addEventListener("click", async () => {
      await withBusy(async () => {
        state.systemStatus = await api("/api/status");
        notice("Server stats updated");
      });
    });
  }

  const answerForm = document.querySelector("#answer-form");
  if (answerForm) {
    answerForm.addEventListener("submit", async (event) => {
      event.preventDefault();
      const answer = fieldValue("answer");
      if (!answer) return;
      await withBusy(async () => {
        await api(`/api/game/${encodeURIComponent(state.gameId)}/answers`, { method: "POST", body: { answer } });
        notice("Answer sent");
        await refreshGame();
      });
    });
  }

  document.querySelectorAll("[data-action]").forEach((button) => {
    button.addEventListener("click", async () => runAction(button.dataset.action));
  });

  document.querySelectorAll("[data-vote]").forEach((button) => {
    button.addEventListener("click", async () => {
      await withBusy(async () => {
        await api(`/api/game/${encodeURIComponent(state.gameId)}/vote`, { method: "POST", body: { answerUUID: button.dataset.vote } });
        setStoredVote(button.dataset.vote);
        notice("Vote recorded");
        await refreshGame();
      });
    });
  });
}

async function runAction(action) {
  const actions = {
    start: { path: "/start", message: "Game started" },
    startVoting: { path: "/startVoting", message: "Voting started" },
    reveal: { message: "Votes revealed", reveal: true },
    next: { path: "/next", message: "Next round started", clearRound: true },
    finish: { path: "/finish", message: "Game finished", stop: true },
    leave: { path: "/leave", message: "Left game", leave: true },
    logout: { logout: true },
    home: { local: true },
  };
  const selected = actions[action];
  if (!selected) return;

  if (needsEmergencyConfirmation(action) && !confirmEmergencyAction(action)) return;

  await withBusy(async () => {
    if (selected.local) {
      leaveGame();
      return;
    }
    if (selected.logout) {
      await api("/api/logout", { method: "POST" });
      notice("Logged out");
      logoutUser();
      return;
    }
    if (selected.reveal) {
      await reveal(true);
    } else {
      await api(`/api/game/${encodeURIComponent(state.gameId)}${selected.path}`, { method: "POST" });
    }
    if (selected.leave) {
      notice(selected.message);
      leaveGame();
      return;
    }
    if (selected.clearRound) {
      clearStoredVote();
      state.answers = [];
      state.revealed = [];
    }
    if (selected.stop) stopTimers();
    notice(selected.message);
    await refreshGame();
  });
}

async function enterGame(gameId, replace) {
  state.gameId = gameId;
  localStorage.setItem("nip.gameId", gameId);
  if (!replace && location.pathname !== `/game/${gameId}`) {
    history.pushState(null, "", `/game/${gameId}`);
  }
  renderStage("loading");
  await refreshGame();
  if (!state.gameId) return;
  loadStoredVote();
  startTimers();
}

function leaveGame() {
  clearStoredVote();
  state.gameId = "";
  state.status = null;
  state.answers = [];
  state.revealed = [];
  state.votedAnswerUUID = "";
  state.menuOpen = false;
  localStorage.removeItem("nip.gameId");
  stopTimers();
  if (location.pathname !== "/") history.pushState(null, "", "/");
  renderStage();
}

function logoutUser() {
  clearStoredVote();
  state.userUUID = "";
  state.username = "";
  state.gameId = "";
  state.status = null;
  state.answers = [];
  state.revealed = [];
  state.votedAnswerUUID = "";
  state.menuOpen = false;
  localStorage.removeItem("nip.userUUID");
  localStorage.removeItem("nip.username");
  localStorage.removeItem("nip.gameId");
  stopTimers();
  if (location.pathname !== "/") history.pushState(null, "", "/");
  renderStage();
}

function startTimers() {
  stopTimers();
  state.pollTimer = window.setInterval(refreshGame, 3000);
}

function stopTimers() {
  clearInterval(state.pollTimer);
}

async function refreshGame() {
  if (!state.gameId || !state.userUUID) {
    renderStage();
    return;
  }
  try {
    state.status = await api(`/api/game/${encodeURIComponent(state.gameId)}/status`, { quiet: true });
    if (state.status.status === 2) {
      returnToChooseGame("Game finished");
      return;
    }
    loadStoredVote();
    await loadPhaseData();
    renderStage();
  } catch (error) {
    if (isUnauthorized(error)) return;
    if (isGameNotFound(error)) {
      returnToChooseGame("Game is no longer available");
      return;
    }
    renderStage("error", error.message);
  }
}

async function loadPhaseData() {
  const roundStatus = currentRoundStatus();
  if (["voting", "revealed"].includes(roundStatus)) {
    try {
      const data = await api(`/api/game/${encodeURIComponent(state.gameId)}/answers`, { quiet: true });
      state.answers = data.answers || [];
    } catch (error) {
      if (isGameNotFound(error)) throw error;
      state.answers = [];
    }
  }
  if (roundStatus === "revealed") {
    try {
      const data = await api(`/api/game/${encodeURIComponent(state.gameId)}/reveal`, { quiet: true });
      state.revealed = data.answers || [];
    } catch (error) {
      if (isGameNotFound(error)) throw error;
      state.revealed = [];
    }
  }
}

async function reveal(sendTrigger) {
  const method = sendTrigger ? "POST" : "GET";
  const data = await api(`/api/game/${encodeURIComponent(state.gameId)}/reveal`, { method, quiet: !sendTrigger });
  state.revealed = data.answers || [];
  renderStage();
}

function renderStage(forcedStage, detail) {
  const stage = forcedStage || resolveStage();
  const renderers = {
    profile: renderProfile,
    choose: renderChooseGame,
    loading: renderLoading,
    lobby: renderLobby,
    answering: renderAnswering,
    voting: renderVoting,
    reveal: renderReveal,
    finished: renderFinished,
    error: () => renderError(detail),
  };
  stageCard.innerHTML = `${renderers[stage]()}${systemStatusWidget()}`;
  bindStageEvents();
}

function resolveStage() {
  if (!state.userUUID) return "profile";
  if (!state.gameId) return "choose";
  if (!state.status) return "loading";
  const roundStatus = currentRoundStatus();
  if (!state.status.round) return "lobby";
  if (roundStatus === "voting") return "voting";
  if (roundStatus === "revealed") return "reveal";
  return "answering";
}

function renderProfile() {
  return `
    <p class="eyebrow">Step 1</p>
    <h1>Nobody is Perfect</h1>
    <p class="lead">Start with your player name. The secure session cookie is handled by the server.</p>
    <form id="profile-form" class="stack">
      <label>Your name<input id="username" autocomplete="nickname" required maxlength="32" placeholder="Ada" value="${escapeAttr(state.username)}"></label>
      <button class="primary" type="submit" ${busyAttr()}>Start session</button>
    </form>
  `;
}

function renderChooseGame() {
  const suggestedGame = state.gameId || "";
  return `
    <p class="eyebrow">Step 2</p>
    <h1>Choose game</h1>
    <p class="lead">Playing as <strong>${escapeHTML(state.username)}</strong>. Create a new table or join an invite.</p>
    <div class="button-row">
      <button id="create-game" class="primary" ${busyAttr()}>Create game</button>
      <button id="join-game-toggle" type="button" aria-expanded="${suggestedGame ? "true" : "false"}" aria-controls="join-form" ${busyAttr()}>Join game</button>
    </div>
    <form id="join-form" class="stack join-panel" ${suggestedGame ? "" : "hidden"}>
      <label>Game ID<input id="join-game-id" autocomplete="off" placeholder="chug.value.funds" value="${escapeAttr(suggestedGame)}"></label>
      <button type="submit" ${busyAttr()}>Join this game</button>
    </form>
    ${detailsMenu([logoutButton()])}
  `;
}

function renderLoading() {
  return `
    <p class="eyebrow">Loading</p>
    <h1>${escapeHTML(state.gameId || "Game")}</h1>
    <p class="lead">Fetching the current game stage.</p>
  `;
}

function renderLobby() {
  const status = state.status || {};
  const isOwner = status.gameMasterUUID === state.userUUID;
  return `
    ${gameHeader("Lobby", "Invite players")}
    <p class="lead">Share the game ID and wait for everyone to join.</p>
    ${invitePanel()}
    ${statRow()}
    ${isOwner ? `<button class="primary" data-action="start" ${busyAttr()}>Start game</button>` : `<p class="waiting">Waiting for the host to start.</p>`}
    ${detailsMenu([playerList(), gameActions(isOwner)])}
  `;
}

function renderAnswering() {
  const status = state.status || {};
  const isRoundMaster = status.roundMasterUUID === state.userUUID;
  const canLead = canLeadRound();
  const primary = isRoundMaster
    ? `<p class="waiting">You are the round master. Wait for the players, then start voting.</p>${canLead ? `<button class="primary" data-action="startVoting" ${busyAttr()}>Start voting</button>` : ""}`
    : `<form id="answer-form" class="stack"><label>Your answer<textarea id="answer" rows="5" required placeholder="Write your best fake answer"></textarea></label><button class="primary" type="submit" ${busyAttr()}>Send answer</button></form>`;
  return `
    ${gameHeader(`Round ${status.round}`, "Answering")}
    ${statRow()}
    ${primary}
    ${detailsMenu([playerList(), canLead && !isRoundMaster ? leaderActions("answering") : "", gameActions(isOwner())])}
  `;
}

function renderVoting() {
  const status = state.status || {};
  const isRoundMaster = status.roundMasterUUID === state.userUUID;
  const canLead = canLeadRound();
  const body = isRoundMaster
    ? `<p class="waiting">You are the round master. Watch the votes come in, then reveal.</p>${canLead ? `<button class="primary" data-action="reveal" ${busyAttr()}>Reveal votes</button>` : ""}`
    : answerChoices();
  return `
    ${gameHeader(`Round ${status.round}`, "Voting")}
    ${statRow()}
    ${body}
    ${detailsMenu([playerList(), canLead && !isRoundMaster ? `<button class="danger" data-action="reveal" ${busyAttr()}>Emergency reveal votes</button>` : "", gameActions(isOwner())])}
  `;
}

function renderReveal() {
  const canLead = canLeadRound();
  return `
    ${gameHeader(`Round ${(state.status || {}).round}`, "Reveal")}
    ${revealedAnswers()}
    ${canLead ? `<button class="primary" data-action="next" ${busyAttr()}>Next round</button>` : `<p class="waiting">Waiting for the next round.</p>`}
    ${detailsMenu([statRow(), playerList(), gameActions(isOwner())])}
  `;
}

function renderFinished() {
  return `
    ${gameHeader("Finished", "Game over")}
    <p class="lead">This game has ended.</p>
    <button class="primary" data-action="home">Back to start</button>
  `;
}

function renderError(detail) {
  return `
    <p class="eyebrow">Problem</p>
    <h1>Could not load game</h1>
    <p class="lead">${escapeHTML(detail || "Something went wrong.")}</p>
    <button class="primary" data-action="home">Back to start</button>
  `;
}

function gameHeader(label, title) {
  return `
    <p class="eyebrow">${escapeHTML(label)}</p>
    <h1>${escapeHTML(title)}</h1>
    <div class="game-code"><span>Game ID</span><strong>${escapeHTML(state.gameId)}</strong></div>
  `;
}

function statRow() {
  const status = state.status || {};
  return `
    <div class="stats">
      <div><span>Players</span><strong>${status.playerCount || 0}</strong></div>
      <div><span>Answers</span><strong>${status.receivedAnswers || 0}</strong></div>
      <div><span>Votes</span><strong>${status.receivedVotes || 0}</strong></div>
    </div>
  `;
}

function systemStatusWidget() {
  const status = state.systemStatus;
  return `
    <section class="system-status" aria-label="Server statistics">
      <button id="system-status" class="info-button" type="button" ${busyAttr()}>i Server stats</button>
      ${status ? `
        <div class="system-status-panel">
          <div><span>Games</span><strong>${status.games || 0}</strong></div>
          <div><span>Players</span><strong>${status.players || 0}</strong></div>
          <div><span>Online</span><strong>${status.onlinePlayers || 0}</strong></div>
        </div>
      ` : ""}
    </section>
  `;
}

function answerChoices() {
  if (state.answers.length === 0) return `<p class="waiting">Answers are not available yet.</p>`;
  return `<div class="answers">${state.answers.map((answer) => `
    <article class="answer-card ${isStoredVote(answer.answerUUID) ? "selected-vote" : ""}">
      <div><strong>${escapeHTML(answer.label || "?")}</strong>${isStoredVote(answer.answerUUID) ? voteLabel() : ""}<p>${escapeHTML(answer.answer || "")}</p>${answer.username ? `<small>${escapeHTML(answer.username)}</small>` : ""}</div>
      <button data-vote="${escapeAttr(answer.answerUUID)}" ${busyAttr()}>${isStoredVote(answer.answerUUID) ? "Your vote" : "Vote"}</button>
    </article>
  `).join("")}</div>`;
}

function revealedAnswers() {
  if (state.revealed.length === 0) return `<p class="waiting">Reveal data is not available yet.</p>`;
  return `<div class="answers">${state.revealed.map((answer) => {
    const votes = (answer.votes || []).map((vote) => vote.username).join(", ") || "no votes";
    return `
      <article class="answer-card revealed ${isStoredVote(answer.answerUUID) ? "selected-vote" : ""}">
        <div><strong>${escapeHTML(answer.label || "?")} by ${escapeHTML(answer.username || "unknown")}</strong>${isStoredVote(answer.answerUUID) ? voteLabel() : ""}<p>${escapeHTML(answer.answer || "")}</p><small>Votes: ${escapeHTML(votes)}</small></div>
      </article>
    `;
  }).join("")}</div>`;
}

function playerList() {
  const status = state.status || {};
  const users = status.users || [];
  if (users.length === 0) return `<p class="muted">No players yet.</p>`;
  return `<ol class="players">${users.map((player) => {
    const flags = [];
    if (player.userUUID === status.gameMasterUUID) flags.push("host");
    if (player.userUUID === status.roundMasterUUID) flags.push("round master");
    return `<li class="player"><span><strong>${escapeHTML(player.username || "Player")}</strong><small class="${player.online ? "online" : "offline"}">${player.online ? "online" : "offline"}</small></span><span class="badge">${escapeHTML(flags.join(" · ") || "player")}</span></li>`;
  }).join("")}</ol>`;
}

function leaderActions(roundStatus) {
  if (roundStatus === "answering") return `<button data-action="startVoting" ${busyAttr()}>Start voting</button>`;
  return "";
}

function gameActions(showFinish) {
  return `
    <button data-action="leave" ${busyAttr()}>Leave game</button>
    ${logoutButton()}
    ${showFinish ? `<button class="danger" data-action="finish" ${busyAttr()}>Finish game</button>` : ""}
  `;
}

function logoutButton() {
  return `<button data-action="logout" ${busyAttr()}>Logout</button>`;
}

function copyLinkButton() {
  return `<button id="copy-link" type="button">Copy invite link</button>`;
}

function invitePanel() {
  const url = inviteLink();
  return `
    <section class="invite-panel" aria-label="Invite players">
      <img class="qr-code" src="${escapeAttr(qrCodeURL(url))}" alt="QR code for joining game ${escapeAttr(state.gameId)}">
      <div class="invite-actions">
        <p class="muted">Scan the QR code or share the invite link.</p>
        <a class="invite-url" href="${escapeAttr(url)}">${escapeHTML(url)}</a>
        <div class="button-row">
          ${copyLinkButton()}
          <button id="share-link" class="primary" type="button">Share invite</button>
        </div>
      </div>
    </section>
  `;
}

function inviteLink() {
  return `${location.origin}/game/${state.gameId}`;
}

function qrCodeURL(value) {
  return `https://api.qrserver.com/v1/create-qr-code/?size=220x220&margin=12&data=${encodeURIComponent(value)}`;
}

function voteLabel() {
  return `<span class="vote-label">Your vote</span>`;
}

function detailsMenu(parts) {
  const content = parts.filter(Boolean).join("");
  if (!content) return "";
  return `
    <section class="action-menu ${state.menuOpen ? "open" : ""}">
      <button id="menu-toggle" class="menu-toggle" type="button" aria-expanded="${state.menuOpen ? "true" : "false"}" aria-controls="menu-panel">
        <span class="hamburger" aria-hidden="true"><span></span><span></span><span></span></span>
        <span>Menu</span>
      </button>
      <div id="menu-panel" class="menu-panel" ${state.menuOpen ? "" : "hidden"}>${content}</div>
    </section>
  `;
}

async function api(path, options = {}) {
  const init = {
    method: options.method || "GET",
    credentials: "same-origin",
    headers: {},
  };
  if (options.body) {
    init.headers["Content-Type"] = "application/json";
    init.body = JSON.stringify(options.body);
  }
  const response = await fetch(path, init);
  const text = await response.text();
  const data = text ? JSON.parse(text) : {};
  if (!response.ok) {
    const error = new Error(data.error || `Request failed with ${response.status}`);
    error.status = response.status;
    if (isUnauthorized(error)) {
      await discardInvalidSession();
      error.handled = true;
    }
    throw error;
  }
  return data;
}

async function withBusy(callback) {
  if (state.busy) return;
  state.busy = true;
  renderStage();
  try {
    await callback();
  } catch (error) {
    if (!error.handled) notice(error.message);
  } finally {
    state.busy = false;
    renderStage();
  }
}

function notice(text) {
  message.textContent = text;
  message.hidden = false;
  clearTimeout(notice.timer);
  notice.timer = setTimeout(() => { message.hidden = true; }, 3500);
}

function currentRoundStatus() {
  return (state.status || {}).roundStatus || "lobby";
}

function isOwner() {
  return (state.status || {}).gameMasterUUID === state.userUUID;
}

function canLeadRound() {
  const status = state.status || {};
  return status.gameMasterUUID === state.userUUID || status.roundMasterUUID === state.userUUID;
}

function returnToChooseGame(reason) {
  clearStoredVote();
  state.gameId = "";
  state.status = null;
  state.answers = [];
  state.revealed = [];
  state.votedAnswerUUID = "";
  state.menuOpen = false;
  localStorage.removeItem("nip.gameId");
  stopTimers();
  if (location.pathname !== "/") history.pushState(null, "", "/");
  renderStage();
  if (reason) notice(reason);
}

function isGameNotFound(error) {
  return error && error.status === 404;
}

function isUnauthorized(error) {
  return error && error.status === 401;
}

async function discardInvalidSession() {
  await fetch("/api/logout", { method: "POST", credentials: "same-origin" }).catch(() => {});
  logoutUser();
  notice("Session expired. Please start again.");
}

function needsEmergencyConfirmation(action) {
  const status = state.status || {};
  if (action === "finish") return true;
  return action === "reveal" && status.gameMasterUUID === state.userUUID && status.roundMasterUUID !== state.userUUID;
}

function confirmEmergencyAction(action) {
  const messages = {
    reveal: "Only reveal votes as the host in an emergency. Reveal votes now?",
    finish: "Finishing ends the game for everyone. Finish the game now?",
  };
  return window.confirm(messages[action] || "Continue?");
}

function loadStoredVote() {
  state.votedAnswerUUID = localStorage.getItem(voteStorageKey()) || "";
}

function setStoredVote(answerUUID) {
  state.votedAnswerUUID = answerUUID || "";
  if (state.votedAnswerUUID) {
    localStorage.setItem(voteStorageKey(), state.votedAnswerUUID);
  }
}

function clearStoredVote() {
  localStorage.removeItem(voteStorageKey());
  state.votedAnswerUUID = "";
}

function voteStorageKey() {
  const round = (state.status || {}).round || 0;
  return `nip.vote.${state.gameId}.${state.userUUID}.${round}`;
}

function isStoredVote(answerUUID) {
  return Boolean(answerUUID && state.votedAnswerUUID && answerUUID === state.votedAnswerUUID);
}

function fieldValue(id) {
  const field = document.querySelector(`#${id}`);
  return field ? field.value.trim() : "";
}

function busyAttr() {
  return state.busy ? "disabled" : "";
}

function escapeHTML(value) {
  return String(value).replace(/[&<>'"]/g, (char) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", "'": "&#39;", '"': "&quot;" }[char]));
}

function escapeAttr(value) {
  return escapeHTML(value);
}

init();
