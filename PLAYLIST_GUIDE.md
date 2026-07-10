# tui-tube Playlist Building Guide

> Context doc for Claude (Bumblebee mode). Read this before touching playlists.

---

## Key Facts

- **MCP DB**: `/home/dev/.local/share/tuitube/tuitube.db` — this is what the MCP tools read/write
- **Dev DB**: `/home/dev/Projects/REPOS/tui-tube/tui-tube.db` — separate, don't confuse them
- **Library size**: ~6,400 unique tracks (post-dedup) across 5 stations
- **Schema**: `playlists(id, name, description, created_at)` — description column was added manually
- **Dedup**: Ran July 2026, removed 1627 duplicate rows (cross-station overlap). Keeper priority: favorited → in any playlist → earliest created_at

---

## Stations

| ID | Name |
|----|------|
| `UCB1L0zM9cJW7jNyQIZu3muQ` | 7clouds Chill |
| `jn74jaahvz9e6jf8vz8e90r9ax8614dk` | Chill & aesthetic music |
| `jn77kd9j7pgtf6vyha366jbjxs861b98` | Lofi beats to relax & study |
| `jn766dgek37mc0cx2q2bftmjdd861z1c` | Trap & hip-hop |
| `jn7b0amyyscf740nb188bet3x1860157` | Vibe Only |

---

## MCP Tools

```
mcp__tuitube__list_stations
mcp__tuitube__list_playlists
mcp__tuitube__list_playlist_tracks      { playlist_id }
mcp__tuitube__search_tracks             { query, limit, station_id, favorites_only }
mcp__tuitube__create_playlist           { name }
mcp__tuitube__add_to_playlist           { playlist_id, track_ids[] }
mcp__tuitube__remove_from_playlist      { playlist_id, track_ids[] }
mcp__tuitube__sync_station              { station_id? }  -- omit for all
mcp__tuitube__play_track / stop_playback / get_now_playing
mcp__tuitube__toggle_favorite
```

> No rename_playlist tool — use sqlite3 on the MCP DB directly.

---

## Philosophy

**Vibe over genre.** Nobody opens a player thinking "I want trap music." They think "I want to feel dangerous" or "I need to cry in a controlled environment." Playlists should feel like curated YouTube mood sets, not Spotify genre stations.

- Minimum ~10 tracks per playlist, no hard upper limit
- Every playlist needs a 1–2 line description — make it feel like a cultural moment or a line people just *get*, not a genre label
- No years in descriptions — write the vibe so people who were there recognize it instantly
- Tracks can (and should) appear in multiple playlists — a song isn't owned by one mood
- The goal is for people to pick a vibe, not a genre

---

## All Current Playlists

| ID | Name | Tracks | Description |
|----|------|--------|-------------|
| 1 | Favorites | 9 | The vault. Not the best songs in the library — the ones that hit different every single time, no matter when you press play. |
| 2 | Boot Sequence | 10 | Claude's first cold-start picks — before the playlists had themes or names. Chill, electric, and a little sentimental. Exactly as the machine intended. |
| 3 | Late Night Drive | 37 | The kind of drive you don't plan. It's 1:45am, you're not going home yet, and the city looks different at this hour. Turn it up. Nobody's watching. |
| 4 | Hood Certified | 33 | The playlist that plays itself at every cookout, every pregame, every car with the bass up at a red light. Trap and hip-hop that never needed your approval to run things. |
| 5 | Spanish Heat | 24 | You don't have to speak the language. Your body already does. Reggaeton that turns a Tuesday into a Saturday night and a Saturday night into something you'll talk about on Monday. |
| 6 | NCS Uprising | 34 | Every montage. Every clutch moment. Every 3am grind session since YouTube was young. Electronic music that genuinely believes you are capable of more than you think. |
| 7 | Lofi Study Hall | 53 | You said you'd study for an hour. Three hours later the playlist is still going and you forgot what you were even stressed about. That's the point. |
| 8 | R&B Sundays | 30 | Sunday has a sound and this is it. R&B that makes doing nothing feel intentional. Sade understood. Miguel understood. You understand now too. |
| 9 | Afrobeats Vibes | 40 | Wizkid played a sold-out O2 and the whole crowd knew every word. Tems is on every playlist in every country. Burna Boy won a Grammy. The continent took over and it sounds like this. |
| 10 | Pop Girl Era | 35 | Not a tribute — a receipt. Doja, SZA, Nicki, Megan, Billie, Taylor, Tems, LISA. The women who ran the last five years of music while everyone else was catching up. |
| 11 | 3AM Crying in the Parking Lot | 32 | You said you were fine. Your playlist says otherwise. For the feelings you didn't know you had until a random song found them at 3am in a parking lot. |
| 12 | Main Character Energy | 29 | You're not just in the room. You're the reason there is a room. For every slow-motion entrance, dramatic walk, and moment you silently gave yourself the title role. |
| 13 | Situationship Soundtrack | 25 | Not your boyfriend. Not just a friend. Absolutely ruining your sleep schedule and somehow still getting away with it. This playlist knows what's going on even if you won't say it. |
| 14 | God Pressed Shuffle | 30 | No genre. No logic. No explanation. Just bangers placed here by something greater than an algorithm. Wizkid next to Taylor next to Travis next to Sade. Trust the process. |
| 15 | Villain Arc | 28 | Everybody has a reason. You just stopped explaining yours. Dark, unbothered, and dangerously good at this now. |
| 16 | Hot Girl Walk | 26 | This is your 30 minutes. No texts, no problems, no eye contact. Just you, the road, and the audacity. Confidence walking pace only. |
| 17 | Healing Arc | 27 | You're not over it. You're just better at carrying it. |
| 18 | Serotonin Boost | 28 | Nothing is wrong. Everything is actually okay. You're happy right now and this playlist is going to make sure it stays that way for at least 30 more minutes. |
| 19 | Sunday Reset | 25 | Laundry's going. Candle's lit. You're not okay but you're being productive about it. The weekly ritual of pretending you have your life together. |
| 20 | Pre-Game Anthems | 20 | You're not going anywhere yet. You're just getting dressed. But the playlist already knows tonight is different. |
| 21 | Plot Armor | 17 | Main characters don't die in act two. This is the soundtrack for people the story won't let lose — Cartoon, TheFatRat, Kendrick, Sia. Epic doesn't cover it. |
| 22 | The Comeback | 16 | You went quiet for a while. People forgot about you. That was the plan. |
| 23 | NPC Mode | 20 | You are just walking to the store. You have no lore. No dramatic backstory. This is the background music of being a side character in your own Tuesday. Owlh-core only. |
| 24 | Enemies to Lovers | 19 | You hate each other. Except when you don't. SZA, Tems, The Weeknd — music for tension that never fully breaks, charged silences, and the line you keep almost crossing. |
| 25 | Midnight Snack | 17 | It is 1am. The fridge is open. You are not hungry, you are just up. Joji, Owlh, Powfu, SZA — soft songs for the hours that don't count. |
| 26 | Rain Day Protocol | 21 | Nowhere to be. Rain on the window. Cigarettes After Sex, Joji, Noah Kahan — music that sounds exactly like watching water streak down glass with a warm drink you forgot to drink. |
| 27 | Tourist Mode | 15 | You just landed somewhere and everything feels possible. Wizkid, Tems, Bad Bunny, Burna Boy, Rauw — the world is big and the playlist knows it. |
| 28 | The Glow Up | 17 | Something shifted and it shows. Doja, SZA, Tems, Billie, The Weeknd — for the version of you that stopped apologizing for taking up space. |
| 29 | Certified Heartbreak | 20 | No coping mechanisms. No silver lining. Just Lil Peep, Juice WRLD, Powfu, and Joji doing what they do. Cry now, process later. |
| 30 | SoundCloud Season | 17 | Before the deals, before the tours, before some of them didn't make it. A bedroom, a mic, and a SoundCloud link in the bio. XXXTentacion, Lil Peep, early Juice WRLD. The rawest era for finding music. |
| 31 | Astroworld Forever | 18 | SICKO MODE on the aux. Pop Smoke running NY. Lil Baby closing out Atlanta. The Weeknd everywhere. Peak streaming, peak going out, peak everything-is-fine energy. The last summer before. |
| 32 | Stuck at Home Hits | 16 | Olivia Rodrigo dropped drivers license and the internet went quiet for three minutes. You found half this playlist from a TikTok. The other half found you at 2am in your childhood bedroom. |
| 33 | The Revenge Era | 20 | Everybody had an era. Taylor had the stadium. Kendrick had the war. SZA had SOS. Billie had LUNCH. The Weeknd danced in flames. Nobody was okay but everyone had a great album about it. |
| 34 | First Date Playlist | 21 | Not trying too hard. Not too quiet either. Giveon, Daniel Caesar, Joji, Lizzy McAlpine, SZA — music cool enough to impress but warm enough that nobody feels like they're being evaluated. |
| 35 | RoadTrip No Signal | 31 | Six hours of drive time, no service, no bad songs allowed. The only playlist that has to work for everyone in the car at 3pm and still be going strong at midnight. |
| 36 | Gym Villain | 20 | You are not here to make friends. You are not here to be comfortable. Travis, Kendrick, 21 Savage, TheFatRat — heavy, dark, and completely unbothered by your feelings about it. |
| 37 | Old Soul | 19 | No trend. No era. Just taste. Sade, Miguel, Giveon, Tems, Cigarettes After Sex — the playlist for people who've always been a little out of time and never minded. |
| 38 | UK On Top | 14 | Central Cee went viral on a freestyle. Dave made a whole film. Stormzy headlined Glastonbury. London stopped waiting for American approval and started running things. This is that moment. |
| 39 | Global Pop Takeover | 19 | BLACKPINK sold out stadiums on every continent. LISA dropped MONEY and the internet broke. ROSALÍA made Grammys feel small. BTS made history at every award show. Pop went global and never came back. |
| 40 | Internet Money Era | 16 | The producer tag dropped and you already knew it was going to be good. Internet Money, Don Toliver, Gunna — the era where every track sounded like a flex and a vibe at the same time. |
| 41 | Café With No Name | 19 | Window seat. Flat white going cold. Lost Frequencies into Kygo into Khalid and somehow two hours disappeared. The café doesn't have a name on the sign but you'll tell people exactly where it is. |
| 42 | Study Hall After Dark | 17 | You opened the notes app, not the textbook. Laufey is on, beabadoobee after. Something about this hour — past midnight, before exhaustion — makes the music feel sharper than the work ever did. |
| 43 | Headphone Diary | 17 | You got new headphones and suddenly heard everything differently. Closer hit different. Paris hit different. That's the playlist — songs you thought you knew, finally heard the way they were meant to land. |
| 44 | Pop Star Treatment | 17 | Three of the most self-possessed women in pop, back to back. Dua Lipa doesn't explain herself. Sabrina Carpenter doesn't apologize. Zara Larsson never needed your approval. Getting-ready music for people who already know. |
| 45 | The Nights | 17 | Avicii told you to live a life you will remember. That dropped and every 19-year-old in a festival crowd pointed at the sky and actually meant it. Martin Garrix was right there beside him. A decade later it hits exactly the same. |
| 46 | Not Sending It | 15 | You typed it. Deleted it. Typed it again. It's 2am and the text is still in your drafts. Put this on instead. |
| 47 | Fake Scenarios | 15 | The conversation didn't go like that. But in your version it did. The ending you wrote in your head at 1am. This is the soundtrack to that one. |
| 48 | 4AM | 15 | It's not 3am anymore. Something shifted. The night went so long it stopped being sad and started being something else. You're still up and you're okay with that. |
| 49 | They Don't Know | 15 | You've noticed the way they laugh at their own jokes. You know the shoes they always wear. They have absolutely no idea. This playlist gets it. |
| 50 | Everyone's Here | 16 | You're by yourself. That's fine. Put this on and the room stops feeling empty. Some songs just do that. |

---

## Playlist Ideas Queue (not yet built)

| Name | Vibe | Notes |
|------|------|-------|
| **iPhone 4 Summer** | 2010-2014 era — YOLO, early Drake, Nicki peak, early EDM | Library thin on this era — mostly has Drake/Nicki newer stuff. Search Flo Rida, Pitbull, David Guetta if revisiting |
| **Deep Latin** | KAROL G, Feid, Shakira, Rauw Alejandro — deeper cuts beyond the hits | Spanish Heat covers the bangers; this is for people who know the B-sides |
| **Keyword Seed** | Start with a phrase ("empty parking lot at 3am") and build purely on association | Experimental — test how discourse-trained vibes translate into track selection |

---

## Tag System

Tags are stored as a JSON array in the `tags TEXT` column on the `playlists` table. Every playlist has 2–4 tags across three dimensions:

| Dimension | Values |
|-----------|--------|
| **Mood** (pick 1–2) | `sad` `happy` `confident` `nostalgic` `romantic` `hype` `healing` |
| **Energy** (pick 1) | `low` `mid` `high` |
| **Setting** (pick 0–2) | `late-night` `study` `drive` `gym` `party` `pregame` `solo` |

Use case: Convex web app can filter playlists by tag — show me all `sad + low`, show me all `party`, etc. MCP agent can also use tags to suggest a playlist from a mood keyword.

---

## Description Rules

When writing descriptions, ask: *would someone who lived through this moment feel seen?*

- No genre labels ("trap and hip-hop", "lo-fi beats") — describe the *feeling* or *moment*
- No years — write the cultural reference so people who were there recognize it without a date
- Artist name-drops are fine but only when they *illustrate* the vibe, not just list who's on it
- First or second person ("you", "it's 1am") hits harder than third person description
- One strong specific image beats two generic sentences

---

## Workflow

1. `mcp__tuitube__search_tracks` with artist/keyword (single term works best)
2. Pick tracks that fit the vibe — check IDs, avoid already-used ones if possible
3. `mcp__tuitube__create_playlist` → get ID
4. `mcp__tuitube__add_to_playlist` with array of IDs
5. Write description directly to MCP DB:
   ```bash
   sqlite3 /home/dev/.local/share/tuitube/tuitube.db \
     "UPDATE playlists SET description = '...' WHERE id = X;"
   ```
6. Use `''` to escape apostrophes in SQL strings

---

## Useful Track IDs (frequently pulled, good anchors)

### Emotional Anchors
- Heather - Conan Gray: `jh7e8cayqfxvm5c8vbqp4zqbxd86149w`
- ceilings - Lizzy McAlpine: `jh70n1fzcvwvfratytgqb3krdn861w30`
- Stick Season - Noah Kahan: `jh704ddswjdx16bs1m6m1h225n860w4r`
- drivers license - Olivia Rodrigo: `jh7cbfwgdrcc4w8vacrr44hgj1860mh5`
- Tejano Blue - Cigarettes After Sex: `jh71q4bhmha3hrhvp244bbqd1n8608cx`
- Run - Joji: `jh704m1myyvtfw4hbt9cwwqdp5861f66`
- Atlantis - Seafret: `jh70djrgxtpj21atm63e3391zn860gf8`
- Falling Down - Lil Peep & XXXTENTACION: `jh72cas12t1jq030jewt49c3cd861895`

### Confidence / Banger Anchors
- Unstoppable - Sia: `jh70dt1w6h8p7k9956kvf1fgv9860w04`
- Cruel Summer - Taylor Swift: `jh76rwk5w09dkt06zyb00r8d85861mj9`
- Streets - Doja Cat: `jh71aka7xyta5hw1hx3cqyhz8x861t7t`
- GOSSIP - Måneskin: `jh702ymzyxknnbp7zjbv4agxjs861yxv`
- Abracadabra - Lady Gaga: `jh70728h5ackjhtrafs8yay0qd860h7j`
- WAP - Cardi B: `jh700bkws359rrr8vhszgmtvy1861eht`
- SICKO MODE - Travis Scott: `jh7091vv0pejrjqbb6g2arkbj986063k`

### Afrobeats Anchors
- Essence ft. Tems - Wizkid: `jh71vrpdef6gczgme43e38rwqs8602r0`
- Joro - Wizkid: `989d8de52bc09adb41362d3ebceb018a`
- Fall - Davido: `42d72a966455c4b82c2c890e43236670`
- Last Last - Burna Boy: `jh727rv9y074kghkthpxnqyz41860dgs`
- Gangsta - Tems: `jh705fybk4rv1a9fzpwex2bf99860nde`
- Lagos Love - Tems: `jh78xfqaknr0q3r01vempehg1x8607qr`

### Chill / Lofi Anchors
- Why We Lose - Cartoon: `jh7apqzg30976nx9y4gw1fn0998618tj`
- Ark - Ship Wrek: `jh739ye7ye6029kbmpfbgnp821861g28`
- Sweet Disposition - Lost Frequencies: `jh7bx470023vxb8vshx8f08tx98607z4`
- facetime and chill - Owlh: `jh7ca40pkta69tm0s8zshmxkhd860brn`
- Good Night - Owlh: `jh793c15mmqfc4hmvb05q37v4d8619q2`
- Alone - Marshmello: `jh705b489vecvy5pgkwf7ft4gh860rtp`

### R&B / Soul Anchors
- No Ordinary Love - Sade: `jh711enf1drf2pvk50kmq9k4ph8617ee`
- Sure Thing - Miguel: `jh71kat9e82cs7t3j982qaqxth860jzw`
- Snooze - SZA: `jh70jb5e4dazj1txpefhmk4wxs861342`
- Good Days - SZA: `jh71980wapwkkvczqbtmwbk9tx86183d`
- Always - Daniel Caesar: `fa81edec6d528e2b402c1c005075ffc5`
- Under The Influence - Chris Brown: `jh7exxn903wtcm19rna0xt46vn861abw`

---

## Web App (Convex + Clerk) — Next Phase

The plan is a voting interface where users vote on what plays next from a queue. The playlist `description` field is already in the DB and ready to surface as playlist card copy. Key data shape that Convex will need to sync:

- `playlists`: id, name, description, track_count
- `tracks`: id, song_title, artist, youtube_id, station_id, is_favorite
- `playlist_tracks`: playlist_id, track_id, added_at
- Vote queue: which playlist is active, current track, next candidates, vote counts

The MCP server already exposes `get_now_playing`, `play_track`, `stop_playback` which the web app backend can call through to control playback.
