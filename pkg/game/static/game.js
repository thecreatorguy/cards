// Imported Constants
const JSON_DATA = JSON.parse(document.getElementById('json-data').innerHTML)
const BASE_PATH = JSON_DATA.base_path
const LOBBY = window.location.href.split("?lobby=")[1];
const IS_HOST = LOBBY === "";

// View Constants
const SetupView = "state";
const LobbyView = "lobby";
const FirstGenView = "first_gen"
const GameView = "game";

// Message Code Constants
const PingCode = "ping";
const SetupCode = "setup";
const HostGameCode = "host_game";
const JoinGameCode = "join_game";
const UpdateCode = "update";
const UpdateLobbySettingsCode = "update_lobby_settings";
const StartGameCode = "start_game";
const FirstGenCode = "first_gen";

const YourTurnCode = "your_turn";
const PlayCardCode = "play_card";
const DrawCardsCode = "draw_cards";
const DiscardCardsCode = "discard_cards";
const DrawPreludesCode = "draw_preludes";
const PlayPreludeCode = "play_prelude";
const DoneTurnCode = "done_turn";
const PassCode = "pass";
const BetweenGensCode = "between_gens";

// State Constants
const InLobbyState = "in_lobby";
const FirstGenState = "first_gen";
const InGenState = "in_gen";
const BetweenGensState = "between_gens";

function makeCorporation(source) {
    let c = document.createElement("div");
    c.classList.add("card-brief");

    let topbar = document.createElement("div");
    topbar.classList.add("top-bar");
    topbar.innerHTML += `<div class="card-type"><div class="corporation">CORPORATION</div></div>`;
    if (source.tags) {
        for (let i = 0; i < source.tags.length; i++) {
            c.innerHTML += `<div class="tag tag${i+1} tag-${source.tags[i]}"></div>`;
        }
    }
    c.appendChild(topbar);

    c.innerHTML += `<div class="title">${source.name}</div>`;
    
    return c;
}

function makePrelude(source) {
    let p = document.createElement("div");
    p.classList.add("card-brief");
    p.innerHTML += "<div class=\"prelude\">PRELUDE</div>";
    p.innerHTML += `<div class="prelude-name">${source.name}</div>`;
    if (source.tags) {
        for (let i = 0; i < source.tags.length; i++) {
            p.innerHTML += `<div class="tag tag${i+1} tag-${source.tags[i]}"></div>`;
        }
    }
    return p;
}

function makeCard(source) {
    let c = document.createElement("div");
    c.classList.add("card-brief");
    // c.innerHTML += "<div class=\"corporation\">CORPORATION</div>";
    c.innerHTML += `<div class="card-name">${source.name}</div>`;
    if (source.tags) {
        for (let i = 0; i < source.tags.length; i++) {
            c.innerHTML += `<div class="tag tag${i+1} tag-${source.tags[i]}"></div>`;
        }
    }
    
    return c;
}


// Controller
let TMController = {

    init() {
        // Init connection
        const urlSplit = window.location.href.split("/");
        const protocol = urlSplit[0] == "http:" ? "ws" : "wss";
        const hostAndPort = urlSplit[2];

        TMController.conn = new WebSocket(protocol + "://" + hostAndPort + BASE_PATH + "/game/websocket");
        TMController.conn.onmessage = e => {
            console.log(e)
            TMController.receive(JSON.parse(e.data))
        };
        TMController.conn.onerror = e => console.log(e); // TODO: close message? redirect to lobby?
        TMController.conn.onclose = e => console.log(e);
    },

    send(msg) {
        TMController.conn.send(JSON.stringify(msg))
    },

    receive(msg) {
        if (msg.code != PingCode) console.log(msg);
        switch(msg.code) {
        case PingCode:
            TMController.send({code: PingCode});
            break;
        case SetupCode:
            TMController.view(SetupView);
            break;
        case UpdateCode:
            TMController.game = msg.content.game;
            TMController.hand = msg.content.hand;

            let g = TMController.game
            if (g.state == InLobbyState) {
                TMController.view(LobbyView);
                TMController.updateLobby(Object.assign({}, g.settings, {players: g.players}))
            } else {
                TMController.view(GameView);
            }
            break;
        case FirstGenCode:
            TMController.view(FirstGenView);
            console.log(msg.content)
            TMController.updateFirstGen(msg.content.corporations, msg.content.preludes, msg.content.cards);
            break;
        case YourTurnCode:
            
        default:
            
        }
    },

    view(view) {
        if (TMController.currentView == view) {
            return
        }
        TMController.currentView = view;


        document.querySelectorAll("main > div").forEach(d => d.hidden = true);

        switch (view) {
        case SetupView:
            document.getElementById("setup-view").hidden = false;
            TMController.initSetup();
            break;
        case LobbyView:
            document.getElementById("lobby-view").hidden = false;
            TMController.initLobby();
            break;
        case FirstGenView:
            document.getElementById("first-gen-view").hidden = false;
            break;
        case GameView:
            document.getElementById("game-view").hidden = false;
            TMController.initGame();
            break;
        }
    },

    //-----------------------------------------------------------
    //------------------------  Setup   -------------------------
    //-----------------------------------------------------------

    initSetup() {
        if (TMController.doneSetup) {
            return;
        }
        TMController.doneSetup = true;

        if (IS_HOST) {
            document.getElementById("submit-setup").onclick = function() {
                const lobbyName = document.getElementById("lobby-name-input").value;
                const nickname = document.getElementById("nickname-input").value;
                TMController.send({code: HostGameCode, content: {
                    lobby_name: lobbyName,
                    nickname: nickname
                }});
            };
        } else {
            document.getElementById("lobbyname").hidden = true;
            document.getElementById("lobby-name-input").required = false;

            document.getElementById("submit-setup").onclick = function() {
                const nickname = document.getElementById("nickname-input").value;
                TMController.send({code: JoinGameCode, content: {
                    lobby_name: LOBBY,
                    nickname: nickname
                }});
            };
        }
    },

    //-----------------------------------------------------------
    //------------------------  Lobby   -------------------------
    //-----------------------------------------------------------

    initLobby() {
        if (TMController.doneLobby) {
            return;
        }
        TMController.doneLobby = true;

        if (IS_HOST) {
            const corpCardsInput = document.getElementById("corp-cards-input");
            corpCardsInput.disabled = false;
            corpCardsInput.addEventListener("click", _ => {
                TMController.send({code: UpdateLobbySettingsCode, content: {
                    use_corporate_cards: corpCardsInput.checked 
                }});
            });

            const preludeCardsInput = document.getElementById("prelude-cards-input");
            preludeCardsInput.disabled = false;
            preludeCardsInput.addEventListener("click", _ => {
                TMController.send({code: UpdateLobbySettingsCode, content: {
                    use_prelude_cards: preludeCardsInput.checked 
                }});
            });

            const startGameButton = document.getElementById("start-game");
            startGameButton.hidden = false;
            startGameButton.addEventListener("click", _ => {
                TMController.send({code: StartGameCode});
            });
        }
    },

    updateLobby(values) {
        const corpCardsInput = document.getElementById("corp-cards-input");
        corpCardsInput.checked = values.use_corporate_cards;

        const preludeCardsInput = document.getElementById("prelude-cards-input");
        preludeCardsInput.checked = values.use_prelude_cards;
        
        const playerList = document.getElementById("player-list");
        playerList.innerHTML = "";
        let items = [];
        let buttons = [];
        for (let p of values.players) {
            let pItem = document.createElement("li");
            playerList.append(pItem);
            items.push(pItem);
            pItem.innerHTML += `<span>${p.nickname}</span>`;
            if (IS_HOST) {
                let swapButton = document.createElement("button");
                pItem.append(swapButton);
                swapButton.innerText = "Swap";
                buttons.push(swapButton);
            }
        }
        if (IS_HOST) {
            let sourceIndex = -1;
            for (let i = 0; i < buttons.length; i++) {
                buttons[i].addEventListener("click", function() {
                    if (sourceIndex == -1) {
                        items[i].classList.add("source-player");
                        sourceIndex = i;
                    } else {
                        items[sourceIndex].classList.remove("source-player");
                        if (sourceIndex != i) {
                            TMController.send({code: UpdateLobbySettingsCode, content: {
                                player_swap_index_1: sourceIndex,
                                player_swap_index_2: i
                            }});
                        }
                        sourceIndex = -1;
                    }
                });
            }
        }
    },

    //-----------------------------------------------------------
    //----------------------  First Gen   -----------------------
    //-----------------------------------------------------------

    updateFirstGen(corporations, preludes, cards) {
        const corporationssDiv = document.querySelector("#first-gen-view .corporations");
        corporationssDiv.innerHTML = "";
        corporations.forEach(c => corporationssDiv.append(makeCorporation(c)));

        const preludesDiv = document.querySelector("#first-gen-view .preludes");
        preludesDiv.innerHTML = "";
        preludes.forEach(p => preludesDiv.append(makePrelude(p)));

        const cardsDiv = document.querySelector("#first-gen-view .cards");
        cardsDiv.innerHTML = "";
        cards.forEach(c => cardsDiv.append(makeCard(c)));
    },

    //-----------------------------------------------------------
    //-----------------------  In Game   ------------------------
    //-----------------------------------------------------------

    initGame() {
        if (TMController.doneGame) {
            return;
        }
        TMController.doneGame = true;

        
    },
};

TMController.init();