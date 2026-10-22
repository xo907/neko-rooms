// Room extras: raised hands and what is playing, plus reactions shared with
// the app. Injected into neko room pages, talks to the
// live channel of the room with the ticket that came with the join link.
//
// Nothing sits on the video: what is playing is in the middle of the top
// bar, raised hands in neko's bottom bar next to its controls. Reactions use neko's own emotes: the ones sent here go
// to the app, the ones sent from the app float up like neko's.
(function () {
  var C = __NR_CONFIG__;
  var KEY = 'nr-ticket:' + C.room;
  var params = new URLSearchParams(location.search);
  var ticket = params.get('nrt');
  try {
    if (ticket) sessionStorage.setItem(KEY, ticket);
    else ticket = sessionStorage.getItem(KEY);
  } catch (e) { /* storage blocked */ }
  if (!ticket || !window.EventSource) return;

  var base = C.api + 'public/live/' + encodeURIComponent(C.room) + '/';
  var me = null, hands = [], playing = '';
  var ui = {};

  //
  // neko client
  //

  function vm() { var el = document.getElementById('neko'); return el && el.__vue__; }
  function acc() { var v = vm(); return v && v.$accessor; }
  function myName() {
    try { var n = acc().user.member.displayname; if (n) return n; } catch (e) { /* not connected */ }
    return params.get('usr') || '';
  }
  function hosting() { try { return !!acc().remote.hosting; } catch (e) { return false; } }
  function hostName() {
    try { var a = acc(), m = a.user.members[a.remote.id]; return m ? m.displayname : ''; } catch (e) { return ''; }
  }
  function memberByName(name) {
    try {
      var ms = acc().user.members;
      for (var id in ms) if (ms[id] && ms[id].displayname === name) return ms[id];
    } catch (e) { /* not connected */ }
    return null;
  }

  // neko emote names <-> emoji, read from neko's own stylesheet, where each
  // emote is a twemoji image named after its code points
  var toEmoji = {}, toEmote = {};
  // without the variation selector, apps and neko differ there
  function plain(e) { return e.split(String.fromCharCode(0xfe0f)).join(''); }
  function readEmotes() {
    for (var i = 0; i < document.styleSheets.length; i++) {
      var rules;
      try { rules = document.styleSheets[i].cssRules; } catch (e) { continue; }
      if (!rules) continue;
      for (var j = 0; j < rules.length; j++) {
        var r = rules[j], m = r.selectorText && /^\.emote\.([a-z0-9-]+)$/.exec(r.selectorText);
        if (!m) continue;
        var f = /\/([0-9a-f-]+)\.[^/]*\.svg/.exec(r.style.backgroundImage || '');
        if (!f) continue;
        try {
          var emoji = String.fromCodePoint.apply(null, f[1].split('-').map(function (h) { return parseInt(h, 16); }));
          toEmoji[m[1]] = emoji;
          toEmote[emoji] = m[1];
          toEmote[plain(emoji)] = m[1];
        } catch (e) { /* not a code point */ }
      }
    }
  }

  // emotes sent here also go to the app
  function hookEmotes() {
    var v = vm(), client = v && v.$client;
    if (!client || client.__nrHooked) return;
    client.__nrHooked = true;
    var send = client.sendMessage;
    client.sendMessage = function (event, payload) {
      if (event === 'chat/emote' && payload && toEmoji[payload.emote]) {
        post('react', { emoji: toEmoji[payload.emote], via: 'neko' }).catch(function () {});
      }
      return send.apply(this, arguments);
    };
  }

  //
  // server
  //

  function post(path, body) {
    body = body || {};
    body.t = ticket;
    body.name = myName();
    return fetch(base + path, {
      method: 'POST',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json', 'X-Requested-With': 'XMLHttpRequest' },
      body: JSON.stringify(body),
    }).then(function (r) {
      if (!r.ok) return r.text().then(function (t) { throw new Error((t || 'Error ' + r.status).trim()); });
    });
  }

  function connect() {
    var es = new EventSource(base + 'events?t=' + encodeURIComponent(ticket) + '&name=' + encodeURIComponent(myName()));
    es.onmessage = function (e) {
      var ev;
      try { ev = JSON.parse(e.data); } catch (err) { return; }
      handle(ev);
    };
  }

  function handle(ev) {
    switch (ev.type) {
      case 'init':
        me = ev.you;
        hands = ev.hands || [];
        playing = ev.playing || '';
        render();
        break;
      case 'hands': hands = ev.hands || []; renderHands(); break;
      case 'playing': playing = ev.title || ''; renderPlaying(); break;
      case 'reaction': if (ev.via !== 'neko') react(ev.emoji, ev.name); break;
    }
  }

  // a reaction from the app, shown like neko's own emotes
  function react(emoji, name) {
    var type = toEmote[emoji] || toEmote[plain(emoji)];
    try {
      if (type) { acc().chat.newEmote({ type: type }); return; }
    } catch (e) { /* fall back */ }
    fly(emoji, name);
  }

  //
  // view
  //

  function el(tag, cls, text) {
    var e = document.createElement(tag);
    if (cls) e.className = cls;
    if (text != null) e.textContent = text;
    return e;
  }

  var css = '' +
    '.nrx{--a:var(--nr-primary,#2dd4bf);--s:rgba(255,255,255,.07);--l:rgba(255,255,255,.1);font:500 13px/1.4 var(--nr-font,system-ui,sans-serif);color:#fafafa}' +
    '.nrx-dock{display:flex;gap:6px;align-items:center;min-width:0;flex-wrap:nowrap}' +
    '.nrx-dock.over{position:absolute;left:12px;bottom:12px;z-index:30;transition:opacity .3s}' +
    '.nrx-dock.over.idle{opacity:0;pointer-events:none}' +
    '.nrx-dock.over .nrx-btn,.nrx-dock.over .nrx-chip{background:rgba(14,14,17,.78);backdrop-filter:blur(12px)}' +
    '.nrx-btn{position:relative;flex:none;display:inline-flex;align-items:center;justify-content:center;gap:6px;height:32px;min-width:32px;padding:0 11px;border:0;border-radius:99px;background:var(--s);color:inherit;font:inherit;cursor:pointer;box-shadow:inset 0 0 0 1px var(--l);transition:background .18s,transform .18s}' +
    '.nrx-btn:hover{background:rgba(255,255,255,.12)}.nrx-btn:active{transform:scale(.96)}' +
    '.nrx-btn.on{background:var(--a);color:#06110f}' +
    '.nrx-badge{position:absolute;top:-5px;right:-5px;min-width:17px;height:17px;padding:0 4px;border-radius:9px;background:var(--a);color:#06110f;font-size:10.5px;line-height:17px;text-align:center}' +
    '.nrx-chip{display:flex;align-items:center;gap:7px;min-width:0;max-width:260px;height:32px;padding:0 4px 0 11px;border-radius:99px;background:var(--s);box-shadow:inset 0 0 0 1px var(--l)}' +
    '.nrx-chip.solo{padding-right:11px}' +
    '.nrx-chip em{font-style:normal;color:rgba(255,255,255,.55);flex:none}' +
    '.nrx-chip .t{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}' +
    '.nrx-chip button{flex:none;height:24px;padding:0 10px;border:0;border-radius:99px;background:var(--a);color:#06110f;font:600 12px/1 inherit;cursor:pointer;white-space:nowrap}' +
    '.nrx-np{position:absolute;left:50%;top:50%;transform:translate(-50%,-50%);z-index:5;max-width:min(420px,40%);background:rgba(255,255,255,.06)}' +
    '.nrx-np em{color:var(--a)}' +
    '.nrx-fly{position:absolute;bottom:20px;z-index:29;pointer-events:none;display:flex;flex-direction:column;align-items:center;animation:nrx-up 3.2s cubic-bezier(.2,.6,.3,1) forwards}' +
    '.nrx-fly b{font-size:30px;line-height:1;filter:drop-shadow(0 4px 10px rgba(0,0,0,.5))}' +
    '.nrx-fly span{margin-top:2px;padding:1px 7px;border-radius:99px;background:rgba(0,0,0,.55);font-size:11px;white-space:nowrap}' +
    '@keyframes nrx-up{0%{transform:translate(0,0) scale(.6);opacity:0}10%{opacity:1;transform:translate(0,-20px) scale(1.1)}100%{transform:translate(var(--dx),-300px) scale(1);opacity:0}}' +
    '.nrx-toast{position:fixed;left:50%;bottom:140px;z-index:1001;transform:translateX(-50%);padding:8px 14px;border-radius:99px;background:rgba(14,14,17,.92);box-shadow:inset 0 0 0 1px var(--l);white-space:nowrap;max-width:80%;overflow:hidden;text-overflow:ellipsis;animation:nrx-fade 3.5s forwards}' +
    '@keyframes nrx-fade{0%,85%{opacity:1}100%{opacity:0}}' +
    '@media (max-width:700px){.nrx-np{display:none!important}}' +
    '@media (max-width:760px){.nrx-btn span.l{display:none}.nrx-chip{max-width:150px}}';

  function mount() {
    var video = document.querySelector('#neko .video-container') || document.querySelector('#neko .video');
    if (!video) return false;
    if (getComputedStyle(video).position === 'static') video.style.position = 'relative';
    ui.video = video;

    var style = el('style');
    style.textContent = css;
    document.head.appendChild(style);

    ui.root = el('div', 'nrx');
    ui.dock = el('div', 'nrx-dock');

    ui.hand = el('button', 'nrx-btn');
    ui.hand.title = 'Raise your hand to get control next';
    ui.hand.onclick = function () {
      post('hand', { up: !mine() }).catch(function (e) { toast(e.message); });
    };

    ui.hands = el('div', 'nrx-chip');
    ui.playing = el('div', 'nrx nrx-chip solo nrx-np');

    ui.dock.appendChild(ui.hand);
    ui.dock.appendChild(ui.hands);
    ui.root.appendChild(ui.dock);

    window.addEventListener('resize', dockIn);

    dockIn();
    return true;
  }

  // dockIn puts the controls in neko's bottom bar, or over the video, out of
  // the way, when the bar is hidden (small screens)
  function dockIn() {
    // what is playing sits in the middle of neko's top bar
    var header = document.querySelector('#neko .header-container .header') || document.querySelector('#neko .header');
    if (header && ui.playing.parentNode !== header) {
      if (getComputedStyle(header).position === 'static') header.style.position = 'relative';
      header.appendChild(ui.playing);
    }

    var bar = document.querySelector('#neko .room-container .room-menu .settings');
    var inBar = bar && bar.offsetParent !== null;
    var target = inBar ? bar : ui.video;
    if (ui.root.parentNode === target) return;
    ui.dock.classList.toggle('over', !inBar);
    target.appendChild(ui.root);
    if (!inBar) idle();
  }

  // over the video the controls fade out while nobody moves
  var idleTimer = 0;
  function idle() {
    ui.dock.classList.remove('idle');
    clearTimeout(idleTimer);
    idleTimer = setTimeout(function () {
      if (ui.dock.classList.contains('over')) ui.dock.classList.add('idle');
    }, 3000);
  }
  ['mousemove', 'touchstart'].forEach(function (t) {
    document.addEventListener(t, function () { if (ui.dock && ui.dock.classList.contains('over')) idle(); }, { passive: true });
  });

  function mine() {
    if (!me) return false;
    for (var i = 0; i < hands.length; i++) if (hands[i].id === me.id) return true;
    return false;
  }

  function render() {
    renderPlaying();
    renderHands();
  }

  function renderPlaying() {
    if (!ui.playing) return;
    ui.playing.innerHTML = '';
    ui.playing.style.display = playing ? '' : 'none';
    if (!playing) return;
    ui.playing.title = playing;
    ui.playing.appendChild(el('em', '', 'Now playing'));
    ui.playing.appendChild(el('span', 't', playing));
  }

  function renderHands() {
    if (!ui.hand) return;
    var up = mine();
    ui.hand.className = 'nrx-btn' + (up ? ' on' : '');
    ui.hand.innerHTML = '<span>✋</span><span class="l">' + (up ? 'Lower hand' : 'Raise hand') + '</span>';
    if (hands.length) ui.hand.appendChild(el('span', 'nrx-badge', String(hands.length)));

    // who is next, and a way to hand over control for the person who has it
    ui.hands.innerHTML = '';
    var first = hands[0];
    ui.hands.style.display = first ? '' : 'none';
    if (!first) return;
    var isMe = me && first.id === me.id;
    var giving = hosting() && !isMe;
    ui.hands.className = 'nrx-chip' + (giving ? '' : ' solo');
    ui.hands.appendChild(el('span', 't', (isMe ? 'You are' : first.name + ' is') + ' next' + (hands.length > 1 ? ' · +' + (hands.length - 1) : '')));
    if (giving) {
      var give = el('button', '', 'Give control');
      give.onclick = function () {
        var m = memberByName(first.name);
        try {
          if (m) acc().remote.give(m);
          else acc().remote.release();
        } catch (e) { /* not connected */ }
        post('hand', { up: false, target: first.id }).catch(function () {});
      };
      ui.hands.appendChild(give);
    }
  }

  // fly floats an emoji neko has no emote for
  function fly(emoji, name) {
    if (!ui.video) return;
    var f = el('div', 'nrx-fly nrx');
    f.appendChild(el('b', '', emoji));
    if (name) f.appendChild(el('span', '', name));
    var w = ui.video.clientWidth || 600;
    f.style.left = Math.round(w * 0.55 + Math.random() * w * 0.35) + 'px';
    f.style.setProperty('--dx', Math.round((Math.random() - 0.5) * 120) + 'px');
    ui.video.appendChild(f);
    setTimeout(function () { f.remove(); }, 3300);
  }

  function toast(text) {
    if (!text) return;
    var t = el('div', 'nrx-toast nrx', text);
    document.body.appendChild(t);
    setTimeout(function () { t.remove(); }, 3600);
  }

  //
  // start once the neko client is connected, so names are known
  //

  var tries = 0;
  var timer = setInterval(function () {
    tries++;
    var a = acc();
    var ready = a && a.user && a.user.member && a.user.member.displayname;
    if (!ready && tries < 30) return;
    if (!ui.root && !mount()) { if (tries < 60) return; }
    clearInterval(timer);
    if (!ui.root) return;
    readEmotes();
    hookEmotes();
    connect();

    // hands follow control: getting it lowers your hand, the person in
    // control sees the give button
    var v = vm();
    if (v && v.$store) {
      v.$store.watch(function (s) { return s.remote && s.remote.id; }, function () {
        if (hosting() && mine()) post('hand', { up: false }).catch(function () {});
        renderHands();
      });
    }
    // neko re-renders its bar on layout changes
    setInterval(dockIn, 2000);
  }, 1000);
})();
