#!/usr/bin/env python3
"""Kordon cloud: the anonymous user count and the Telegram alerts bot.

Runs behind nginx at https://efimovdev.ru/kordon/api/ (access log off for it).
Everything it keeps is in one SQLite file:

  pings  - day, version, OS, how many pings. No IP, no install id: a Kordon
           that agreed to be counted sends {"v": "1.3.0", "os": "darwin"} once a day.
  chats  - a Telegram chat id and the SHA-256 of the secret its Kordon holds.
           Alert texts are passed on to Telegram and not stored.

Python 3.10+, standard library only. Settings come from the environment:
  KORDON_BOT_TOKEN   Telegram bot token (without it the bot part is off)
  KORDON_OWNER_CHAT  chat id that may ask the bot for /stats
  KORDON_DB          path to the SQLite file (default ./kordon.db)
  KORDON_PORT        port on 127.0.0.1 (default 8095)
"""

import hashlib
import json
import os
import secrets
import sqlite3
import threading
import time
import urllib.error
import urllib.parse
import urllib.request
from datetime import datetime, timezone
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

TOKEN = os.environ.get("KORDON_BOT_TOKEN", "")
OWNER = os.environ.get("KORDON_OWNER_CHAT", "")
DB = os.environ.get("KORDON_DB", "kordon.db")
PORT = int(os.environ.get("KORDON_PORT", "8095"))
PREFIX = "/kordon/api"
CODE_TTL = 15 * 60  # a link code from Kordon is valid for 15 minutes
SEND_LIMIT = 30  # alerts per hour per Kordon

OSES = {"darwin", "windows", "linux"}
db_lock = threading.Lock()
bot_name = ""
codes = {}  # code -> (secret hash, created), kept in memory only
sent = {}  # secret hash -> [send times] for the rate limit


def conn():
    c = sqlite3.connect(DB)
    c.execute("CREATE TABLE IF NOT EXISTS pings (day TEXT, version TEXT, os TEXT, n INTEGER, PRIMARY KEY (day, version, os))")
    c.execute("CREATE TABLE IF NOT EXISTS chats (secret TEXT PRIMARY KEY, chat INTEGER, lang TEXT, created TEXT)")
    return c


def today():
    return datetime.now(timezone.utc).strftime("%Y-%m-%d")


def h(secret):
    return hashlib.sha256(secret.encode()).hexdigest()


def tg(method, **params):
    if not TOKEN:
        raise RuntimeError("no bot token")
    data = json.dumps(params).encode()
    req = urllib.request.Request(f"https://api.telegram.org/bot{TOKEN}/{method}", data=data, headers={"Content-Type": "application/json"})
    with urllib.request.urlopen(req, timeout=60) as r:
        return json.loads(r.read())["result"]


TEXTS = {
    "ru": {
        "linked": "Kordon подключён. Напишу сюда, когда VPN отвалится и программы встанут на паузу, и когда всё вернётся.\n\nОтключить: /stop или в панели Kordon.",
        "hello": "Это бот Kordon - kill switch для приложений.\n\nЧтобы получать сюда уведомления, откройте панель Kordon → Настройки → Telegram → «Подключить».\n\nKordon: https://efimovdev.ru/kordon/",
        "expired": "Ссылка устарела. Нажмите «Подключить» в панели Kordon ещё раз.",
        "stopped": "Готово, больше не пишу. Подключить снова можно в панели Kordon.",
    },
    "en": {
        "linked": "Kordon is connected. I'll write here when the VPN drops and apps are paused, and when it's back.\n\nTo turn it off: /stop or in the Kordon panel.",
        "hello": "This is the Kordon bot - a kill switch for apps.\n\nTo get alerts here, open the Kordon panel → Settings → Telegram → Connect.\n\nKordon: https://efimovdev.ru/kordon/",
        "expired": "This link has expired. Press Connect in the Kordon panel again.",
        "stopped": "Done, no more messages. You can connect again in the Kordon panel.",
    },
}


def text(lang, key):
    return TEXTS["en" if lang == "en" else "ru"][key]


def stats_text():
    with db_lock, conn() as c:
        rows = c.execute("SELECT day, SUM(n) FROM pings GROUP BY day ORDER BY day DESC LIMIT 14").fetchall()
        by_os = c.execute("SELECT os, SUM(n) FROM pings WHERE day = ? GROUP BY os", (today(),)).fetchall()
        by_v = c.execute("SELECT version, SUM(n) FROM pings WHERE day = ? GROUP BY version ORDER BY 2 DESC", (today(),)).fetchall()
        chats = c.execute("SELECT COUNT(*) FROM chats").fetchone()[0]
    lines = ["Kordon, пользователи в день (кто согласился на счётчик):"]
    lines += [f"{d}: {n}" for d, n in rows] or ["пока никого"]
    if by_os:
        lines.append("Сегодня по системам: " + ", ".join(f"{o} {n}" for o, n in by_os))
    if by_v:
        lines.append("Сегодня по версиям: " + ", ".join(f"{v} {n}" for v, n in by_v))
    lines.append(f"Подключили Telegram: {chats}")
    return "\n".join(lines)


def bot_loop():
    global bot_name
    if not TOKEN:
        return
    while not bot_name:
        try:
            bot_name = tg("getMe")["username"]
        except Exception:
            time.sleep(10)
    offset = 0
    while True:
        try:
            updates = tg("getUpdates", offset=offset, timeout=50, allowed_updates=["message"])
        except Exception:
            time.sleep(5)
            continue
        for u in updates:
            offset = u["update_id"] + 1
            m = u.get("message") or {}
            chat = (m.get("chat") or {}).get("id")
            body = (m.get("text") or "").strip()
            lang = ((m.get("from") or {}).get("language_code") or "ru")[:2]
            if not chat:
                continue
            try:
                handle(chat, body, lang)
            except Exception:
                pass


def handle(chat, body, lang):
    if body.startswith("/start"):
        code = body[6:].strip()
        if not code:
            tg("sendMessage", chat_id=chat, text=text(lang, "hello"), disable_web_page_preview=True)
            return
        item = codes.pop(code, None)
        if not item or time.time() - item[1] > CODE_TTL:
            tg("sendMessage", chat_id=chat, text=text(lang, "expired"))
            return
        with db_lock, conn() as c:
            c.execute("INSERT OR REPLACE INTO chats VALUES (?, ?, ?, ?)", (item[0], chat, lang, today()))
        tg("sendMessage", chat_id=chat, text=text(lang, "linked"))
    elif body == "/stop":
        with db_lock, conn() as c:
            c.execute("DELETE FROM chats WHERE chat = ?", (chat,))
        tg("sendMessage", chat_id=chat, text=text(lang, "stopped"))
    elif body == "/id":
        tg("sendMessage", chat_id=chat, text=f"chat id: {chat}")
    elif body == "/stats" and OWNER and str(chat) == OWNER:
        tg("sendMessage", chat_id=chat, text=stats_text())
    else:
        tg("sendMessage", chat_id=chat, text=text(lang, "hello"), disable_web_page_preview=True)


class Handler(BaseHTTPRequestHandler):
    server_version = "kordon"

    def log_message(self, *args):  # no access log: nothing about who called
        pass

    def reply(self, code, obj):
        b = json.dumps(obj, ensure_ascii=False).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(b)))
        self.send_header("Cache-Control", "no-store")
        self.end_headers()
        self.wfile.write(b)

    def body(self):
        n = min(int(self.headers.get("Content-Length") or 0), 4096)
        try:
            return json.loads(self.rfile.read(n) or b"{}")
        except Exception:
            return {}

    def secret(self):
        s = self.headers.get("X-Kordon-Secret", "")
        return h(s) if len(s) >= 32 else ""

    def do_GET(self):
        path = urllib.parse.urlparse(self.path).path
        if path == PREFIX + "/health":
            return self.reply(200, {"ok": True, "bot": bool(TOKEN)})
        if path == PREFIX + "/tg/status":
            s = self.secret()
            if not s:
                return self.reply(400, {"error": "no secret"})
            with db_lock, conn() as c:
                linked = c.execute("SELECT 1 FROM chats WHERE secret = ?", (s,)).fetchone() is not None
            return self.reply(200, {"linked": linked})
        self.reply(404, {"error": "not found"})

    def do_POST(self):
        path = urllib.parse.urlparse(self.path).path
        if path == PREFIX + "/ping":
            d = self.body()
            v, os_ = str(d.get("v", ""))[:20], str(d.get("os", ""))
            if os_ not in OSES or not v or not all(ch.isdigit() or ch == "." for ch in v):
                return self.reply(400, {"error": "bad ping"})
            with db_lock, conn() as c:
                c.execute("INSERT INTO pings VALUES (?, ?, ?, 1) ON CONFLICT (day, version, os) DO UPDATE SET n = n + 1", (today(), v, os_))
            return self.reply(200, {"ok": True})
        if not TOKEN:
            return self.reply(503, {"error": "bot is off"})
        s = self.secret()
        if not s:
            return self.reply(400, {"error": "no secret"})
        if path == PREFIX + "/tg/link":
            for k, (_, t) in list(codes.items()):
                if time.time() - t > CODE_TTL:
                    codes.pop(k, None)
            code = secrets.token_urlsafe(12)
            codes[code] = (s, time.time())
            return self.reply(200, {"url": f"https://t.me/{bot_name}?start={code}" if bot_name else ""})
        if path == PREFIX + "/tg/unlink":
            with db_lock, conn() as c:
                c.execute("DELETE FROM chats WHERE secret = ?", (s,))
            return self.reply(200, {"ok": True})
        if path == PREFIX + "/tg/send":
            msg = str(self.body().get("text", ""))[:600]
            now = time.time()
            recent = [t for t in sent.get(s, []) if now - t < 3600]
            if len(recent) >= SEND_LIMIT:
                return self.reply(429, {"error": "too many"})
            with db_lock, conn() as c:
                row = c.execute("SELECT chat FROM chats WHERE secret = ?", (s,)).fetchone()
            if not row:
                return self.reply(404, {"error": "not linked"})
            sent[s] = recent + [now]
            try:
                tg("sendMessage", chat_id=row[0], text=msg)
            except urllib.error.HTTPError as e:
                if e.code == 403:  # the user blocked the bot
                    with db_lock, conn() as c:
                        c.execute("DELETE FROM chats WHERE secret = ?", (s,))
                    return self.reply(404, {"error": "not linked"})
                return self.reply(502, {"error": "telegram"})
            except Exception:
                return self.reply(502, {"error": "telegram"})
            return self.reply(200, {"ok": True})
        self.reply(404, {"error": "not found"})


if __name__ == "__main__":
    conn().close()
    threading.Thread(target=bot_loop, daemon=True).start()
    ThreadingHTTPServer(("127.0.0.1", PORT), Handler).serve_forever()
