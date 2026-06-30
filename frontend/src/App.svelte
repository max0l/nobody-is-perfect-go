<script>
  import { onDestroy, onMount } from "svelte";

  export let initialGameId = "";

  let userUUID = localStorage.getItem("nip.userUUID") || "";
  let username = localStorage.getItem("nip.username") || "";
  let gameId = initialGameId || localStorage.getItem("nip.gameId") || "";
  let status = null;
  let answers = [];
  let revealed = [];
  let votedAnswerUUID = "";
  let systemStatus = null;
  let systemStatusOpen = false;
  let pollTimer = 0;
  let busy = false;
  let menuOpen = false;
  let hostSettingsOpen = false;
  let qrOpen = false;
  let profileName = username;
  let joinGameId = gameId;
  let joinOpen = Boolean(joinGameId);
  let answerDraft = "";
  let message = "";
  let messageTimer = 0;
  let forcedStage = "";
  let errorDetail = "";
  let lastRoundKey = "";
  let answerField;

  $: roundStatus = status?.roundStatus || "lobby";
  $: stage = forcedStage || (!userUUID ? "profile" : !gameId ? "choose" : !status ? "loading" : !status.round ? "lobby" : roundStatus === "verifying" ? "verifying" : roundStatus === "voting" ? "voting" : roundStatus === "revealed" ? "reveal" : "answering");
  $: isGameOwner = status?.gameMasterUUID === userUUID;
  $: canLead = status?.gameMasterUUID === userUUID || status?.roundMasterUUID === userUUID;
  $: currentAnswer = status?.currentAnswer || "";
  $: currentAnswerUUID = status?.currentAnswerUUID || "";
  $: hasEnoughPlayers = (status?.playerCount || 0) >= 3;
  $: allPlayersAnswered = (status?.receivedAnswers || 0) >= (status?.playerCount || 0);
  $: requiredVotes = Math.max((status?.playerCount || 0) - 1, 0);
  $: allEligiblePlayersVoted = (status?.receivedVotes || 0) >= requiredVotes;
  $: stats = [
    { label: "Players", value: status?.playerCount || 0 },
    ...(["answering", "verifying"].includes(roundStatus) && status?.roundMasterUUID === userUUID ? [{ label: "Answers", value: status?.receivedAnswers || 0 }] : []),
    ...(["voting", "revealed"].includes(roundStatus) && status?.roundMasterUUID === userUUID ? [{ label: "Votes", value: status?.receivedVotes || 0 }] : []),
  ];

  onMount(async () => {
    window.addEventListener("popstate", handlePopState);
    if (gameId && userUUID) {
      await enterGame(gameId, false);
    }
  });

  onDestroy(() => {
    window.removeEventListener("popstate", handlePopState);
    stopTimers();
    clearTimeout(messageTimer);
  });

  function handlePopState() {
    const match = location.pathname.match(/^\/game\/([^/]+)$/);
    if (match) {
      enterGame(decodeURIComponent(match[1]), true);
      return;
    }
    gameId = "";
    status = null;
    menuOpen = false;
    hostSettingsOpen = false;
    qrOpen = false;
    joinOpen = false;
    stopTimers();
  }

  async function submitProfile() {
    const nextUsername = profileName.trim();
    if (!nextUsername) return;
    await withBusy(async () => {
      const user = await api("/api/create/user", { method: "POST", body: { username: nextUsername } });
      userUUID = user.userUUID || "";
      username = nextUsername;
      localStorage.setItem("nip.userUUID", userUUID);
      localStorage.setItem("nip.username", username);
      notice(`Session started for ${username}`);
      if (gameId) {
        await joinGame(gameId);
      }
    });
  }

  async function createGame() {
    await withBusy(async () => {
      const game = await api("/api/create/game", { method: "POST" });
      await enterGame(game.gameId, false);
      notice(`Created game ${game.gameId}`);
    });
  }

  async function joinGameFromForm() {
    const nextGameId = joinGameId.trim();
    if (!nextGameId) return;
    await withBusy(async () => {
      await joinGame(nextGameId);
    });
  }

  async function joinGame(nextGameId) {
    try {
      await api(`/api/join/${encodeURIComponent(nextGameId)}`, { method: "POST" });
      await enterGame(nextGameId, false);
      notice(`Joined game ${nextGameId}`);
    } catch (error) {
      if (gameId === nextGameId && !status) clearPendingGame();
      throw error;
    }
  }

  async function enterGame(nextGameId, replace) {
    if (gameId !== nextGameId) {
      qrOpen = false;
      answerDraft = "";
    }
    gameId = nextGameId;
    joinGameId = nextGameId;
    localStorage.setItem("nip.gameId", nextGameId);
    if (!replace && location.pathname !== `/game/${nextGameId}`) {
      history.pushState(null, "", `/game/${nextGameId}`);
    }
    forcedStage = "loading";
    await refreshGame();
    forcedStage = "";
    if (!gameId) return;
    loadStoredVote();
    startTimers();
  }

  async function submitAnswer() {
    const answer = answerDraft.trim();
    if (!answer) return;
    await withBusy(async () => {
      await api(`/api/game/${encodeURIComponent(gameId)}/answers`, { method: "POST", body: { answer } });
      answerDraft = "";
      notice(currentAnswer ? "Answer overwritten" : "Answer sent");
      await refreshGame();
    });
  }

  async function editCurrentAnswer() {
    answerDraft = currentAnswer;
    await Promise.resolve();
    answerField?.focus();
  }

  async function voteForAnswer(answerUUID) {
    await withBusy(async () => {
      await api(`/api/game/${encodeURIComponent(gameId)}/vote`, { method: "POST", body: { answerUUID } });
      setStoredVote(answerUUID);
      notice("Vote recorded");
      await refreshGame();
    });
  }

  async function deleteAnswer(answerUUID) {
    if (!window.confirm("Delete this answer before voting starts?")) return;
    await withBusy(async () => {
      await api(`/api/game/${encodeURIComponent(gameId)}/answers/${encodeURIComponent(answerUUID)}`, { method: "DELETE" });
      answers = answers.filter((answer) => answer.answerUUID !== answerUUID);
      notice("Answer deleted");
      await refreshGame();
    });
  }

  async function runAction(action) {
    const actions = {
      start: { path: "/start", message: "Game started" },
      startVerification: { path: "/startVerification", message: "Verification started" },
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
        await revealVotes(true);
      } else {
        await api(`/api/game/${encodeURIComponent(gameId)}${selected.path}`, { method: "POST" });
      }
      if (selected.leave) {
        notice(selected.message);
        leaveGame();
        return;
      }
      if (selected.clearRound) {
        clearStoredVote();
        answers = [];
        revealed = [];
        answerDraft = "";
      }
      if (selected.stop) stopTimers();
      notice(selected.message);
      await refreshGame();
    });
  }

  async function refreshGame() {
    if (!gameId || !userUUID) return;
    try {
      status = await api(`/api/game/${encodeURIComponent(gameId)}/status`, { quiet: true });
      if (status.status === 2) {
        returnToChooseGame("Game finished");
        return;
      }
      clearDraftOnRoundChange();
      loadStoredVote();
      await loadPhaseData();
      forcedStage = "";
    } catch (error) {
      if (isUnauthorized(error)) return;
      if (isGameNotFound(error)) {
        returnToChooseGame("Game is no longer available");
        return;
      }
      errorDetail = error.message;
      forcedStage = "error";
    }
  }

  async function loadPhaseData() {
    if (["verifying", "voting", "revealed"].includes(currentRoundStatus())) {
      try {
        const data = await api(`/api/game/${encodeURIComponent(gameId)}/answers`, { quiet: true });
        answers = data.answers || [];
      } catch (error) {
        if (isGameNotFound(error)) throw error;
        answers = [];
      }
    }
    if (currentRoundStatus() === "revealed") {
      try {
        const data = await api(`/api/game/${encodeURIComponent(gameId)}/reveal`, { quiet: true });
        revealed = data.answers || [];
      } catch (error) {
        if (isGameNotFound(error)) throw error;
        revealed = [];
      }
    }
  }

  async function revealVotes(sendTrigger) {
    const method = sendTrigger ? "POST" : "GET";
    const data = await api(`/api/game/${encodeURIComponent(gameId)}/reveal`, { method, quiet: !sendTrigger });
    revealed = data.answers || [];
  }

  async function loadSystemStatus() {
    if (systemStatusOpen) {
      systemStatusOpen = false;
      return;
    }
    systemStatusOpen = true;
    if (systemStatus) return;

    await withBusy(async () => {
      systemStatus = await api("/api/status");
      notice("Server stats updated");
    });
  }

  async function copyInviteLink() {
    await navigator.clipboard.writeText(inviteLink());
    notice("Invite link copied");
  }

  async function shareInviteLink() {
    const url = inviteLink();
    try {
      if (navigator.share) {
        await navigator.share({ title: "Join my Nobody is Perfect game", text: `Join game ${gameId}`, url });
        return;
      }
      await navigator.clipboard.writeText(url);
      notice("Invite link copied");
    } catch (error) {
      if (error.name !== "AbortError") notice(error.message);
    }
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
    if (busy) return;
    busy = true;
    try {
      await callback();
    } catch (error) {
      if (!error.handled) notice(error.message);
    } finally {
      busy = false;
    }
  }

  function startTimers() {
    stopTimers();
    pollTimer = window.setInterval(refreshGame, 3000);
  }

  function stopTimers() {
    clearInterval(pollTimer);
  }

  function clearPendingGame() {
    gameId = "";
    joinGameId = "";
    joinOpen = false;
    status = null;
    menuOpen = false;
    hostSettingsOpen = false;
    localStorage.removeItem("nip.gameId");
    if (location.pathname !== "/") history.pushState(null, "", "/");
  }

  function leaveGame() {
    clearStoredVote();
    gameId = "";
    joinGameId = "";
    joinOpen = false;
    status = null;
    answers = [];
    revealed = [];
    votedAnswerUUID = "";
    answerDraft = "";
    menuOpen = false;
    hostSettingsOpen = false;
    qrOpen = false;
    forcedStage = "";
    localStorage.removeItem("nip.gameId");
    stopTimers();
    if (location.pathname !== "/") history.pushState(null, "", "/");
  }

  function logoutUser() {
    clearStoredVote();
    userUUID = "";
    username = "";
    profileName = "";
    gameId = "";
    joinGameId = "";
    joinOpen = false;
    status = null;
    answers = [];
    revealed = [];
    votedAnswerUUID = "";
    answerDraft = "";
    menuOpen = false;
    hostSettingsOpen = false;
    qrOpen = false;
    forcedStage = "";
    localStorage.removeItem("nip.userUUID");
    localStorage.removeItem("nip.username");
    localStorage.removeItem("nip.gameId");
    stopTimers();
    if (location.pathname !== "/") history.pushState(null, "", "/");
  }

  function returnToChooseGame(reason) {
    clearStoredVote();
    gameId = "";
    joinGameId = "";
    joinOpen = false;
    status = null;
    answers = [];
    revealed = [];
    votedAnswerUUID = "";
    answerDraft = "";
    menuOpen = false;
    hostSettingsOpen = false;
    qrOpen = false;
    forcedStage = "";
    localStorage.removeItem("nip.gameId");
    stopTimers();
    if (location.pathname !== "/") history.pushState(null, "", "/");
    if (reason) notice(reason);
  }

  async function discardInvalidSession() {
    await fetch("/api/logout", { method: "POST", credentials: "same-origin" }).catch(() => {});
    logoutUser();
    notice("Session expired. Please start again.");
  }

  function clearDraftOnRoundChange() {
    const nextRoundKey = `${gameId}:${status?.round || 0}:${status?.roundStatus || "lobby"}`;
    if (lastRoundKey && lastRoundKey !== nextRoundKey && status?.roundStatus !== "answering") {
      answerDraft = "";
    }
    lastRoundKey = nextRoundKey;
  }

  function currentRoundStatus() {
    return status?.roundStatus || "lobby";
  }

  function inviteLink() {
    return `${location.origin}/game/${gameId}`;
  }

  function qrCodeURL(value) {
    return `https://api.qrserver.com/v1/create-qr-code/?size=220x220&margin=12&data=${encodeURIComponent(value)}`;
  }

  function voteNames(answer) {
    return (answer.votes || []).map((vote) => vote.username).join(", ") || "no votes";
  }

  function playerFlags(player) {
    const flags = [];
    if (player.userUUID === status?.gameMasterUUID) flags.push("host");
    if (player.userUUID === status?.roundMasterUUID) flags.push("round master");
    return flags.join(" · ") || "player";
  }

  function setStoredVote(answerUUID) {
    votedAnswerUUID = answerUUID || "";
    if (votedAnswerUUID) localStorage.setItem(voteStorageKey(), votedAnswerUUID);
  }

  function loadStoredVote() {
    votedAnswerUUID = localStorage.getItem(voteStorageKey()) || "";
  }

  function clearStoredVote() {
    localStorage.removeItem(voteStorageKey());
    votedAnswerUUID = "";
  }

  function voteStorageKey() {
    return `nip.vote.${gameId}.${userUUID}.${status?.round || 0}`;
  }

  function isStoredVote(answerUUID) {
    return Boolean(answerUUID && votedAnswerUUID && answerUUID === votedAnswerUUID);
  }

  function isCurrentUserAnswer(answerUUID) {
    return Boolean(answerUUID && currentAnswerUUID && answerUUID === currentAnswerUUID);
  }

  function needsEmergencyConfirmation(action) {
    if (action === "finish") return true;
    if (action === "startVerification") return isGameOwner && (!allPlayersAnswered || !hasEnoughPlayers);
    if (action === "startVoting") return isGameOwner && !hasEnoughPlayers;
    if (action === "reveal") return isGameOwner && !allEligiblePlayersVoted;
    if (action === "next") return isGameOwner && !hasEnoughPlayers;
    return false;
  }

  function confirmEmergencyAction(action) {
    const messages = {
      startVerification: "Not all players have submitted an answer or fewer than 3 players are active. Review answers anyway?",
      startVoting: "There are fewer than 3 players. Start voting anyway?",
      reveal: "Not all eligible players have voted. Reveal votes anyway?",
      next: "There are fewer than 3 players. Start the next round anyway?",
      finish: "Finishing ends the game for everyone. Finish the game now?",
    };
    return window.confirm(messages[action] || "Continue?");
  }

  function isGameNotFound(error) {
    return error && error.status === 404;
  }

  function isUnauthorized(error) {
    return error && error.status === 401;
  }

  function notice(text) {
    message = text;
    clearTimeout(messageTimer);
    messageTimer = setTimeout(() => {
      message = "";
    }, 3500);
  }
</script>

<main class="app">
  <section class="card stage-card" aria-live="polite">
    {#if stage === "profile"}
      <p class="eyebrow">Step 1</p>
      <h1>Nobody is Perfect</h1>
      <p class="lead">Start with your player name. The secure session cookie is handled by the server.</p>
      <form class="stack" on:submit|preventDefault={submitProfile}>
        <label>Your name<input bind:value={profileName} autocomplete="nickname" required maxlength="32" placeholder="Ada" /></label>
        <button class="primary" type="submit" disabled={busy}>Start session</button>
      </form>
    {:else if stage === "choose"}
      <p class="eyebrow">Step 2</p>
      <h1>Choose game</h1>
      <p class="lead">Playing as <strong>{username}</strong>. Create a new table or join an invite.</p>
      <div class="button-row">
        <button id="create-game" class="primary" disabled={busy} on:click={createGame}>Create game</button>
        <button type="button" aria-expanded={joinOpen} aria-controls="join-form" disabled={busy} on:click={() => (joinOpen = !joinOpen)}>Join game</button>
      </div>
      <form id="join-form" class="stack join-panel" hidden={!joinOpen} on:submit|preventDefault={joinGameFromForm}>
        <label>Game ID<input bind:value={joinGameId} autocomplete="off" placeholder="chug.value.funds" /></label>
        <button type="submit" disabled={busy}>Join this game</button>
      </form>
      {@render ActionMenu(ChooseMenu)}
    {:else if stage === "loading"}
      <p class="eyebrow">Loading</p>
      <h1>{gameId || "Game"}</h1>
      <p class="lead">Fetching the current game stage.</p>
    {:else if stage === "lobby"}
      {@render GameHeader("Lobby", "Invite players")}
      <p class="lead">Share the game ID and wait for everyone to join.</p>
      {@render InvitePanel()}
      {@render Stats()}
      {#if !isGameOwner}
        <p class="waiting">Waiting for the host to start.</p>
      {:else if (status?.playerCount || 0) < 3}
        <button class="primary" type="button" disabled>Need 3 players to start</button>
        <p class="waiting">Waiting for at least 3 players including the host.</p>
      {:else}
        <button class="primary" disabled={busy} on:click={() => runAction("start")}>Start game</button>
      {/if}
      {@render ActionMenu(LobbyMenu)}
      {@render HostSettings()}
    {:else if stage === "answering"}
      {@render GameHeader(`Round ${status?.round || ""}`, "Answering")}
      {@render Stats()}
      {#if status?.roundMasterUUID === userUUID}
        <p class="waiting">You are the round master. You can submit your own answer, then review all answers.</p>
        {#if !hasEnoughPlayers}
          <p class="waiting">Waiting for at least 3 players to continue.</p>
        {:else if allPlayersAnswered}
          <button class="primary" disabled={busy} on:click={() => runAction("startVerification")}>Review answers</button>
        {:else}
          <p class="waiting">Waiting for all players to submit answers.</p>
        {/if}
      {/if}
      {#if currentAnswer}
        <section class="submitted-answer" aria-label="Your submitted answer">
          <div>
            <span>Your submitted answer</span>
            <p>{currentAnswer}</p>
          </div>
          <button type="button" disabled={busy} on:click={editCurrentAnswer}>Edit</button>
        </section>
      {/if}
      <form class="stack" on:submit|preventDefault={submitAnswer}>
        <label>Your answer<textarea bind:this={answerField} bind:value={answerDraft} rows="5" required placeholder="Write your best fake answer"></textarea></label>
        {#if currentAnswer}<p class="overwrite-note">Sending a new answer will overwrite your previous answer.</p>{/if}
        <button class="primary" type="submit" disabled={busy}>{currentAnswer ? "Overwrite answer" : "Send answer"}</button>
      </form>
      {@render ActionMenu(AnsweringMenu)}
      {@render HostSettings()}
    {:else if stage === "verifying"}
      {@render GameHeader(`Round ${status?.round || ""}`, "Review answers")}
      {@render Stats()}
      {#if status?.roundMasterUUID === userUUID}
        <p class="waiting">Delete answers that should not be included. Deleted answers will not appear during voting.</p>
        {#if answers.length === 0}
          <p class="waiting">No answers are available yet.</p>
        {:else}
          <div class="answers">
            {#each answers as answer (answer.answerUUID)}
              <article class="answer-card">
                <div><strong>{answer.label || "?"} by {answer.username || "unknown"}</strong><p>{answer.answer || ""}</p></div>
                <button class="danger" disabled={busy} on:click={() => deleteAnswer(answer.answerUUID)}>Delete</button>
              </article>
            {/each}
          </div>
        {/if}
        {#if hasEnoughPlayers}
          <button class="primary" disabled={busy} on:click={() => runAction("startVoting")}>Start voting</button>
        {:else}
          <p class="waiting">Waiting for at least 3 players to continue.</p>
        {/if}
      {:else}
        <p class="waiting">The round master is reviewing the answers.</p>
      {/if}
      {@render ActionMenu(VerifyingMenu)}
      {@render HostSettings()}
    {:else if stage === "voting"}
      {@render GameHeader(`Round ${status?.round || ""}`, "Voting")}
      {@render Stats()}
      {#if status?.roundMasterUUID === userUUID}
        <p class="waiting">You are the round master. Watch the votes come in, then reveal.</p>
        {#if allEligiblePlayersVoted}
          <button class="primary" disabled={busy} on:click={() => runAction("reveal")}>Reveal votes</button>
        {:else}
          <p class="waiting">Waiting for all eligible players to vote.</p>
        {/if}
      {:else if answers.length === 0}
        <p class="waiting">Answers are not available yet.</p>
      {:else}
        <div class="answers">
          {#each answers as answer (answer.answerUUID)}
            <article class:selected-vote={isStoredVote(answer.answerUUID)} class="answer-card">
              <div><strong>{answer.label || "?"}</strong>{#if isStoredVote(answer.answerUUID)}<span class="vote-label">Your vote</span>{/if}<p>{answer.answer || ""}</p>{#if answer.username}<small>{answer.username}</small>{/if}</div>
              {#if isCurrentUserAnswer(answer.answerUUID)}<small class="own-answer-note">Your submitted answer</small>{/if}
              <button disabled={busy} on:click={() => voteForAnswer(answer.answerUUID)}>{isStoredVote(answer.answerUUID) ? "Your vote" : "Vote"}</button>
            </article>
          {/each}
        </div>
      {/if}
      {@render ActionMenu(VotingMenu)}
      {@render HostSettings()}
    {:else if stage === "reveal"}
      {@render GameHeader(`Round ${status?.round || ""}`, "Reveal")}
      {#if revealed.length === 0}
        <p class="waiting">Reveal data is not available yet.</p>
      {:else}
        <div class="answers">
          {#each revealed as answer (answer.answerUUID)}
            <article class:selected-vote={isStoredVote(answer.answerUUID)} class="answer-card revealed">
              <div><strong>{answer.label || "?"} by {answer.username || "unknown"}</strong>{#if isStoredVote(answer.answerUUID)}<span class="vote-label">Your vote</span>{/if}<p>{answer.answer || ""}</p><small>Votes: {voteNames(answer)}</small></div>
            </article>
          {/each}
        </div>
      {/if}
      {#if status?.roundMasterUUID === userUUID && hasEnoughPlayers}
        <button class="primary" disabled={busy} on:click={() => runAction("next")}>Next round</button>
      {:else if status?.roundMasterUUID === userUUID}
        <p class="waiting">Waiting for at least 3 players to continue.</p>
      {:else}
        <p class="waiting">Waiting for the next round.</p>
      {/if}
      {@render ActionMenu(RevealMenu)}
      {@render HostSettings()}
    {:else if stage === "error"}
      <p class="eyebrow">Problem</p>
      <h1>Could not load game</h1>
      <p class="lead">{errorDetail || "Something went wrong."}</p>
      <button class="primary" on:click={() => runAction("home")}>Back to start</button>
    {/if}

    <section class="system-status" aria-label="Server statistics">
      <button class="info-button" type="button" aria-expanded={systemStatusOpen} disabled={busy} on:click={loadSystemStatus}>Server stats</button>
      {#if systemStatusOpen && systemStatus}
        <div class="system-status-panel">
          <div><span>Games</span><strong>{systemStatus.games || 0}</strong></div>
          <div><span>Players</span><strong>{systemStatus.players || 0}</strong></div>
          <div><span>Online</span><strong>{systemStatus.onlinePlayers || 0}</strong></div>
        </div>
      {/if}
    </section>
  </section>

  {#if message}<p class="toast" role="status" aria-live="polite">{message}</p>{/if}
</main>

{#snippet GameHeader(label, title)}
  <p class="eyebrow">{label}</p>
  <h1>{title}</h1>
  <div class="game-code"><span>Game ID</span><strong>{gameId}</strong></div>
{/snippet}

{#snippet Stats()}
  <div class="stats">
    {#each stats as stat}
      <div><span>{stat.label}</span><strong>{stat.value}</strong></div>
    {/each}
  </div>
{/snippet}

{#snippet InvitePanel()}
  <section class="invite-panel" aria-label="Invite players">
    <div class="invite-actions">
      <p class="muted">Share the invite link or open the QR code for nearby players.</p>
      <a class="invite-url" href={inviteLink()}>{inviteLink()}</a>
      <div class="button-row">
        <button type="button" aria-expanded={qrOpen} aria-controls="qr-code" on:click={() => (qrOpen = !qrOpen)}>{qrOpen ? "Hide QR code" : "Show QR code"}</button>
        <button type="button" on:click={copyInviteLink}>Copy invite link</button>
        <button class="primary" type="button" on:click={shareInviteLink}>Share invite</button>
      </div>
    </div>
    {#if qrOpen}<img id="qr-code" class="qr-code" src={qrCodeURL(inviteLink())} alt={`QR code for joining game ${gameId}`} />{/if}
  </section>
{/snippet}

{#snippet ActionMenu(children)}
  <section class:open={menuOpen} class="action-menu">
    <button class="menu-toggle" type="button" aria-expanded={menuOpen} aria-controls="menu-panel" on:click={() => (menuOpen = !menuOpen)}>
      <span class="hamburger" aria-hidden="true"><span></span><span></span><span></span></span>
      <span>Menu</span>
    </button>
    <div id="menu-panel" class="menu-panel" hidden={!menuOpen}>{@render children()}</div>
  </section>
{/snippet}

{#snippet HostSettings()}
  {#if isGameOwner && gameId && status}
    <section class:open={hostSettingsOpen} class="action-menu">
      <button class="menu-toggle" type="button" aria-expanded={hostSettingsOpen} aria-controls="host-settings-panel" on:click={() => (hostSettingsOpen = !hostSettingsOpen)}>
        <span class="hamburger" aria-hidden="true"><span></span><span></span><span></span></span>
        <span>Host settings</span>
      </button>
      <div id="host-settings-panel" class="menu-panel" hidden={!hostSettingsOpen}>
        {#if roundStatus === "answering" && status?.roundMasterUUID !== userUUID}
          <button disabled={busy} on:click={() => runAction("startVerification")}>{allPlayersAnswered && hasEnoughPlayers ? "Review answers" : "Emergency review answers"}</button>
        {/if}
        {#if roundStatus === "answering" && status?.roundMasterUUID === userUUID && (!allPlayersAnswered || !hasEnoughPlayers)}
          <button disabled={busy} on:click={() => runAction("startVerification")}>Emergency review answers</button>
        {/if}
        {#if roundStatus === "verifying" && (status?.roundMasterUUID !== userUUID || !hasEnoughPlayers)}
          <button disabled={busy} on:click={() => runAction("startVoting")}>{hasEnoughPlayers ? "Start voting" : "Emergency start voting"}</button>
        {/if}
        {#if roundStatus === "voting" && (status?.roundMasterUUID !== userUUID || !allEligiblePlayersVoted)}
          <button class:danger={!allEligiblePlayersVoted} disabled={busy} on:click={() => runAction("reveal")}>{allEligiblePlayersVoted ? "Reveal votes" : "Emergency reveal votes"}</button>
        {/if}
        {#if roundStatus === "revealed" && (status?.roundMasterUUID !== userUUID || !hasEnoughPlayers)}
          <button disabled={busy} on:click={() => runAction("next")}>{hasEnoughPlayers ? "Next round" : "Emergency next round"}</button>
        {/if}
        <button class="danger" disabled={busy} on:click={() => runAction("finish")}>Finish game</button>
      </div>
    </section>
  {/if}
{/snippet}

{#snippet ChooseMenu()}
  <button disabled={busy} on:click={() => runAction("logout")}>Logout</button>
{/snippet}

{#snippet LobbyMenu()}
  {@render PlayerList()}
  {@render GameActions()}
{/snippet}

{#snippet AnsweringMenu()}
  {@render PlayerList()}
  {@render GameActions()}
{/snippet}

{#snippet VerifyingMenu()}
  {@render PlayerList()}
  {@render GameActions()}
{/snippet}

{#snippet VotingMenu()}
  {@render PlayerList()}
  {@render GameActions()}
{/snippet}

{#snippet RevealMenu()}
  {@render Stats()}
  {@render PlayerList()}
  {@render GameActions()}
{/snippet}

{#snippet PlayerList()}
  {#if !status?.users || status.users.length === 0}
    <p class="muted">No players yet.</p>
  {:else}
    <ol class="players">
      {#each status.users as player (player.userUUID)}
        <li class="player"><span><strong>{player.username || "Player"}</strong><small class:online={player.online} class:offline={!player.online}>{player.online ? "online" : "offline"}</small></span><span class="badge">{playerFlags(player)}</span></li>
      {/each}
    </ol>
  {/if}
{/snippet}

{#snippet GameActions()}
  <button disabled={busy} on:click={() => runAction("leave")}>Leave game</button>
  <button disabled={busy} on:click={() => runAction("logout")}>Logout</button>
{/snippet}
