// Room extras: raised hands, rewind, subtitles and what is playing, plus
// reactions shared with the app. Injected into neko room pages, talks to the
// live channel of the room with the ticket that came with the join link.
//
// Nothing sits on the video: what is playing is in the middle of the top
// bar, raised hands in neko's bottom bar next to its controls. Reactions use neko's own emotes: the ones sent here go
// to the app, the ones sent from the app float up like neko's.
(function () {
  var C = __NR_CONFIG__;

  function vm() { var el = document.getElementById('neko'); return el && el.__vue__; }
  function acc() { var v = vm(); return v && v.$accessor; }

  // the rewind recorder is in the room like a viewer, nobody should see it:
  // not in the member list, and no join message with the chat sound
  (function hideRecorder() {
    var hidden = {};
    var tries = 0;
    var timer = setInterval(function () {
      var v = vm(), a = acc();
      if (!v || !v.$store || !a || !a.chat) { if (++tries > 120) clearInterval(timer); return; }
      clearInterval(timer);
      function scrub() {
        var st = v.$store.state, ms = st.user && st.user.members;
        if (ms) {
          for (var id in ms) {
            if (ms[id] && ms[id].displayname === C.recorder) { hidden[id] = true; v.$delete(ms, id); }
          }
        }
        var h = st.chat && st.chat.history;
        if (h && h.some(function (m) { return hidden[m.id]; })) {
          st.chat.history = h.filter(function (m) { return !hidden[m.id]; });
        }
      }
      try {
        var newMessage = a.chat.newMessage;
        a.chat.newMessage = function (m) {
          if (m && hidden[m.id]) return;
          return newMessage.apply(this, arguments);
        };
      } catch (e) { /* read only, scrub takes it out afterwards */ }
      v.$store.subscribe(scrub);
      scrub();
    }, 250);
  })();

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
        ccJoin();
        break;
      case 'hands': hands = ev.hands || []; renderHands(); break;
      case 'playing': playing = ev.title || ''; renderPlaying(); break;
      case 'reaction': if (ev.via !== 'neko') react(ev.emoji, ev.name); break;
      case 'caption': if (ev.lang === cc.lang) caption(ev.text); break;
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
    '.nrx-cc{position:absolute;left:50%;bottom:7%;z-index:35;transform:translateX(-50%);display:flex;flex-direction:column;align-items:center;gap:4px;width:max-content;max-width:min(860px,86%);pointer-events:none;text-align:center}' +
    '.nrx-cc span{padding:3px 10px;border-radius:6px;background:rgba(0,0,0,.78);color:#fff;font:500 clamp(14px,2.1vw,24px)/1.35 var(--nr-font,system-ui,sans-serif);text-shadow:0 1px 2px #000}' +
    '.nrx-menu{position:fixed;z-index:1002;display:flex;flex-direction:column;min-width:170px;max-height:min(360px,60vh);overflow:auto;padding:5px;border-radius:12px;background:rgba(14,14,17,.96);backdrop-filter:blur(12px);box-shadow:inset 0 0 0 1px var(--l),0 12px 32px rgba(0,0,0,.5)}' +
    '.nrx-menu button{padding:8px 12px;border:0;border-radius:8px;background:none;color:inherit;font:inherit;text-align:left;cursor:pointer}' +
    '.nrx-menu button:hover{background:rgba(255,255,255,.08)}.nrx-menu button.on{color:var(--a)}' +
    '.nrx-cc-mark{display:inline-block;padding:0 3px;border:1.5px solid currentColor;border-radius:3px;font:700 10px/1.3 system-ui,sans-serif}' +
    '.nrx-rw{position:absolute;inset:0;z-index:40;display:flex;align-items:center;justify-content:center;background:#000}' +
    '.nrx-rw video{width:100%;height:100%;object-fit:contain;background:#000}' +
    '.nrx-rw-top{position:absolute;top:12px;left:12px;right:12px;display:flex;justify-content:center;pointer-events:none}' +
    '.nrx-rw-badge{display:inline-flex;align-items:center;height:28px;padding:0 12px;border-radius:99px;background:rgba(14,14,17,.78);backdrop-filter:blur(12px);box-shadow:inset 0 0 0 1px var(--l);font-variant-numeric:tabular-nums}' +
    '.nrx-rw-bar{position:absolute;left:12px;right:12px;bottom:12px;display:flex;align-items:center;gap:6px;padding:5px;border-radius:99px;background:rgba(14,14,17,.78);backdrop-filter:blur(12px);box-shadow:inset 0 0 0 1px var(--l)}' +
    '.nrx-rw-bar .nrx-btn{height:30px;min-width:30px;font-variant-numeric:tabular-nums}' +
    '.nrx-rw-range{flex:1;min-width:60px;margin:0 6px;accent-color:var(--a)}' +
    '@media (max-width:520px){.nrx-rw-bar .opt{display:none}}' +
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

    ui.rewind = el('button', 'nrx-btn');
    ui.rewind.innerHTML = '<span>⏪</span><span class="l">Rewind</span>';
    ui.rewind.style.display = 'none';
    ui.rewind.onclick = openRewind;

    ui.cc = el('button', 'nrx-btn');
    ui.cc.innerHTML = '<span class="nrx-cc-mark">CC</span><span class="l">Subtitles</span>';
    ui.cc.style.display = 'none';
    ui.cc.onclick = function (e) { e.stopPropagation(); ccMenu(); };
    ui.playing = el('div', 'nrx nrx-chip solo nrx-np');

    ui.dock.appendChild(ui.cc);
    ui.dock.appendChild(ui.rewind);
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

  //
  // rewind: the last minutes of the room, recorded on the server, played
  // over the live room until you catch up
  //

  var RW_TYPE = 'video/webm; codecs="vp8, opus"';
  var rw = { idx: null, at: 0, player: null };
  rw.ok = !!(window.MediaSource && MediaSource.isTypeSupported && MediaSource.isTypeSupported(RW_TYPE));

  function rwFetch() {
    return fetch(base + 'rewind?t=' + encodeURIComponent(ticket), { credentials: 'same-origin' })
      .then(function (r) { return r.ok ? r.json() : null; })
      .then(function (idx) {
        if (idx) { rw.idx = idx; rw.at = Date.now(); }
        renderRewind();
      })
      .catch(function () { /* try again later */ });
  }
  function rwSegs() { return (rw.idx && rw.idx.segments) || []; }
  // ms on the recording timeline right now
  function rwNow() { return rw.idx ? rw.idx.now + (Date.now() - rw.at) : 0; }
  function rwFirst() { var s = rwSegs(); return s.length ? s[0].start : 0; }
  function rwLast() { var s = rwSegs(); return s.length ? s[s.length - 1].end : 0; }
  function rwAvailable() { return rw.ok && rwSegs().length > 0 && rwLast() - rwFirst() >= 15000; }
  function rwSegFrom(n) {
    var s = rwSegs();
    for (var i = 0; i < s.length; i++) if (s[i].n >= n) return s[i];
    return null;
  }

  function clock(ms) {
    var s = Math.max(0, Math.round(ms / 1000)), m = Math.floor(s / 60);
    s %= 60;
    return m + ':' + (s < 10 ? '0' : '') + s;
  }

  function renderRewind() {
    if (!ui.rewind) return;
    ui.rewind.style.display = rwAvailable() && !rw.player ? '' : 'none';
    if (rwAvailable()) ui.rewind.title = 'See the last ' + clock(rwNow() - rwFirst()) + ' and catch up';
  }

  function openRewind() {
    if (rw.player || !rwAvailable()) return;
    var p = rw.player = { gen: 0, next: 0, loading: false, pending: null, target: null, sb: null, speeds: [1, 1.5, 2], speed: 0 };

    p.box = el('div', 'nrx nrx-rw');
    p.video = el('video');
    p.video.playsInline = true;
    p.box.appendChild(p.video);

    var top = el('div', 'nrx-rw-top');
    p.badge = el('span', 'nrx-rw-badge');
    top.appendChild(p.badge);
    p.box.appendChild(top);

    var bar = el('div', 'nrx-rw-bar');
    p.play = el('button', 'nrx-btn');
    p.play.onclick = function () { if (p.video.paused) p.video.play().catch(function () {}); else p.video.pause(); };
    var back = el('button', 'nrx-btn opt', '-10s');
    back.onclick = function () { jump(-10); };
    var fwd = el('button', 'nrx-btn opt', '+10s');
    fwd.onclick = function () { jump(10); };
    p.rate = el('button', 'nrx-btn', '1×');
    p.rate.title = 'Speed, faster catches up';
    p.rate.onclick = function () {
      p.speed = (p.speed + 1) % p.speeds.length;
      p.video.playbackRate = p.speeds[p.speed];
      p.rate.textContent = p.speeds[p.speed] + '×';
    };
    p.range = el('input', 'nrx-rw-range');
    p.range.type = 'range';
    p.range.step = '0.5';
    p.range.oninput = function () { p.dragging = true; p.badge.textContent = '⏪ ' + clock(rwNow() - p.range.value * 1000) + ' behind'; };
    p.range.onchange = function () { p.dragging = false; to(Number(p.range.value)); };
    var live = el('button', 'nrx-btn on', 'Live');
    live.title = 'Back to the room';
    live.onclick = function () { closeRewind(); };
    [p.play, back, fwd, p.rate, p.range, live].forEach(function (b) { bar.appendChild(b); });
    p.box.appendChild(bar);
    ui.video.appendChild(p.box);

    // the live room goes quiet meanwhile
    p.muted = [];
    document.querySelectorAll('#neko video').forEach(function (v) {
      if (v !== p.video) { p.muted.push([v, v.muted]); v.muted = true; }
    });
    p.keys = function (e) { if (e.key === 'Escape') closeRewind(); };
    document.addEventListener('keydown', p.keys);

    var ms = new MediaSource();
    p.video.src = URL.createObjectURL(ms);
    ms.addEventListener('sourceopen', function () {
      if (rw.player !== p) return;
      try { p.sb = ms.addSourceBuffer(RW_TYPE); } catch (e) { closeRewind('Rewind does not work in this browser'); return; }
      p.sb.mode = 'segments';
      p.sb.addEventListener('updateend', pump);
      // a couple of minutes back to start with
      seek(Math.max(rwFirst(), rwNow() - 120000));
    }, { once: true });

    p.timer = setInterval(tick, 500);
    p.refresh = setInterval(rwFetch, 5000);
    renderRewind();
    tick();
  }

  function closeRewind(msg) {
    var p = rw.player;
    if (!p) return;
    rw.player = null;
    p.gen++;
    clearInterval(p.timer);
    clearInterval(p.refresh);
    document.removeEventListener('keydown', p.keys);
    try { p.video.pause(); } catch (e) { /* gone */ }
    p.video.removeAttribute('src');
    try { p.video.load(); } catch (e) { /* gone */ }
    p.box.remove();
    p.muted.forEach(function (m) { m[0].muted = m[1]; });
    renderRewind();
    if (msg) toast(msg);
  }

  function inBuffer(t) {
    var b = rw.player.sb && rw.player.sb.buffered;
    if (!b) return false;
    for (var i = 0; i < b.length; i++) if (t >= b.start(i) && t < b.end(i)) return true;
    return false;
  }

  // to plays from second t of the timeline, from what is loaded if it can
  function to(t) {
    var p = rw.player;
    if (!p) return;
    if (inBuffer(t)) p.video.currentTime = t;
    else seek(t * 1000);
  }
  function jump(sec) {
    var p = rw.player;
    if (p) to(Math.max(rwFirst() / 1000, p.video.currentTime + sec));
  }

  // seek starts loading over from the segment with time t (ms)
  function seek(t) {
    var p = rw.player;
    if (!p || !p.sb) return;
    var segs = rwSegs(), seg = null;
    for (var i = 0; i < segs.length; i++) if (segs[i].end > t) { seg = segs[i]; break; }
    if (!seg) { closeRewind('You are caught up'); return; }
    p.gen++;
    p.next = seg.n;
    p.target = Math.max(t, seg.start) / 1000;
    p.loading = false;
    p.pending = null;
    try { p.sb.abort(); } catch (e) { /* not open yet */ }
    if (p.sb.buffered.length) {
      try { p.sb.remove(0, Infinity); return; } catch (e) { /* nothing to remove */ }
    }
    pump();
  }

  // pump keeps about 20 seconds loaded ahead and lets go of what was watched
  function pump() {
    var p = rw.player;
    if (!p || !p.sb || p.sb.updating) return;
    if (p.pending) {
      var data = p.pending;
      p.pending = null;
      try { p.sb.appendBuffer(data); } catch (e) { seek(p.video.currentTime * 1000); }
      return;
    }
    var v = p.video, b = p.sb.buffered;

    // after a seek, play from there once it is loaded
    if (p.target != null && b.length) {
      for (var i = 0; i < b.length; i++) {
        if (b.end(i) > p.target) {
          v.currentTime = Math.max(p.target, b.start(i));
          p.target = null;
          v.play().catch(function () { /* paused until a tap */ });
          break;
        }
      }
    }

    if (b.length && v.currentTime - b.start(0) > 90) {
      p.sb.remove(b.start(0), v.currentTime - 30);
      return;
    }
    if (p.loading) return;
    var end = b.length ? b.end(b.length - 1) : 0;
    if (p.target == null && end - v.currentTime > 20) return;

    var seg = rwSegFrom(p.next);
    if (!seg) return; // newer ones come with the next index
    var gen = p.gen;
    p.loading = true;
    fetch(base + 'rewind/' + seg.n + '?t=' + encodeURIComponent(ticket), { credentials: 'same-origin' })
      .then(function (r) {
        if (!r.ok) throw new Error('Error ' + r.status);
        return r.arrayBuffer();
      })
      .then(function (data) {
        if (gen !== p.gen || rw.player !== p) return;
        p.loading = false;
        p.next = seg.n + 1;
        p.pending = data;
        pump();
      })
      .catch(function () {
        if (gen !== p.gen) return;
        // gone in the meantime, go on with the next one
        p.loading = false;
        p.next = seg.n + 1;
      });
  }

  function tick() {
    var p = rw.player;
    if (!p) return;
    var v = p.video, b = p.sb && p.sb.buffered;

    // jump over the times nobody watched, nothing was recorded then
    if (b && b.length && p.target == null && !v.seeking && !v.paused && !inBuffer(v.currentTime + 0.1)) {
      for (var i = 0; i < b.length; i++) if (b.start(i) > v.currentTime) { v.currentTime = b.start(i); break; }
    }
    pump();

    var now = rwNow(), at = v.currentTime * 1000;
    if (p.target != null) at = p.target * 1000;

    // caught up with the recording: back to the room
    var loaded = b && b.length ? b.end(b.length - 1) * 1000 : 0;
    if (p.target == null && !p.loading && !p.pending && !v.paused && loaded && at >= loaded - 700 && !rwSegFrom(p.next)) {
      closeRewind('You are caught up');
      return;
    }

    p.range.min = String(rwFirst() / 1000);
    p.range.max = String(now / 1000);
    if (!p.dragging) {
      p.range.value = String(at / 1000);
      p.badge.textContent = '⏪ ' + clock(now - at) + ' behind';
    }
    p.play.textContent = v.paused ? '▶' : '❚❚';
  }

  //
  // subtitles: what is said in the room, in the language each viewer picks
  //

  var CC_KEY = 'nr-captions';
  var cc = { lang: '', langs: null, lines: [], menu: null };
  try { cc.lang = localStorage.getItem(CC_KEY) || ''; } catch (e) { /* storage blocked */ }

  function ccLanguages() {
    return fetch(base + 'captions?t=' + encodeURIComponent(ticket), { credentials: 'same-origin' })
      .then(function (r) { return r.ok ? r.json() : { languages: [] }; })
      .catch(function () { return { languages: [] }; })
      .then(function (d) {
        cc.langs = d.languages || [];
        renderCC();
        return cc.langs;
      });
  }

  function ccName(code) {
    var l = (cc.langs || []).filter(function (x) { return x.code === code; })[0];
    return l ? l.name : code;
  }

  // ccJoin tells the server the language picked before, after every connect
  function ccJoin() {
    ccLanguages().then(function (langs) {
      if (!cc.lang) return;
      if (langs.some(function (l) { return l.code === cc.lang; })) post('captions', { lang: cc.lang }).catch(function () {});
      else { cc.lang = ''; renderCC(); }
    });
  }

  function ccSet(lang) {
    cc.lang = lang;
    try { localStorage.setItem(CC_KEY, lang); } catch (e) { /* storage blocked */ }
    post('captions', { lang: lang }).catch(function (e) { toast(e.message); });
    cc.lines = [];
    renderCaption();
    renderCC();
    if (lang) toast('Subtitles: ' + ccName(lang) + ', they start with the next words');
  }

  function renderCC() {
    if (!ui.cc) return;
    ui.cc.style.display = cc.langs && cc.langs.length ? '' : 'none';
    ui.cc.className = 'nrx-btn' + (cc.lang ? ' on' : '');
    ui.cc.title = cc.lang ? 'Subtitles: ' + ccName(cc.lang) : 'Subtitles';
  }

  function ccMenu() {
    if (cc.menu) { closeCCMenu(); return; }
    var m = cc.menu = el('div', 'nrx nrx-menu');
    var opts = [{ code: '', name: 'Off' }].concat(cc.langs || []);
    opts.forEach(function (o) {
      var b = el('button', o.code === cc.lang ? 'on' : '', o.name);
      b.onclick = function () { closeCCMenu(); if (o.code !== cc.lang) ccSet(o.code); };
      m.appendChild(b);
    });
    document.body.appendChild(m);
    var r = ui.cc.getBoundingClientRect();
    var h = m.offsetHeight, w = m.offsetWidth;
    m.style.left = Math.max(8, Math.min(r.left, window.innerWidth - w - 8)) + 'px';
    m.style.top = (r.top - h - 8 > 8 ? r.top - h - 8 : r.bottom + 8) + 'px';
    setTimeout(function () { document.addEventListener('click', closeCCMenu, { once: true }); }, 0);
  }
  function closeCCMenu() {
    if (cc.menu) { cc.menu.remove(); cc.menu = null; }
  }

  // the last two lines, each shown for a few seconds
  var CC_SHOW = 7000;
  function caption(text) {
    cc.lines.push({ text: text, at: Date.now() });
    if (cc.lines.length > 2) cc.lines.shift();
    renderCaption();
    setTimeout(renderCaption, CC_SHOW + 50);
  }
  function renderCaption() {
    if (!ui.video) return;
    if (!ui.capt) {
      ui.capt = el('div', 'nrx nrx-cc');
      ui.video.appendChild(ui.capt);
    }
    var now = Date.now();
    cc.lines = cc.lines.filter(function (l) { return now - l.at < CC_SHOW; });
    ui.capt.innerHTML = '';
    cc.lines.forEach(function (l) { ui.capt.appendChild(el('span', '', l.text)); });
    ui.capt.style.display = cc.lang && cc.lines.length ? '' : 'none';
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
    if (rw.ok) {
      rwFetch();
      setInterval(function () { if (!rw.player) rwFetch(); }, 20000);
    }

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
