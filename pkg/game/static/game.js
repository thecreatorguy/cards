const JSON_DATA = JSON.parse(document.getElementById('json-data').innerHTML)
const BASE_PATH = JSON_DATA.base_path

const SetupView = "state";
const LobbyView = "lobby";
const GameView = "game";

const PingCode = "ping";
const SetupCode = "setup";
const HostGameCode = "host_game";
const JoinGameCode = "join_game";
const UpdateCode = "update";
const UpdateLobbySettingsCode = "update_lobby_settings";
const StartGameCode = "start_game";
const FirstGenCode = "first_gen";

const InLobbyState = "in_lobby";
const FirstGenState = "first_gen";
const InGenState = "in_gen";
const BetweenGensState = "between_gens";

const LOBBY = window.location.href.split("?lobby=")[1];
const IS_HOST = LOBBY === "";

let TMController = {

    init() {
        // Init connection
        const urlSplit = window.location.href.split("/");
        const protocol = urlSplit[0] == "http:" ? "ws" : "wss";
        const hostAndPort = urlSplit[2];

        TMController.conn = new WebSocket(protocol + "://" + hostAndPort + BASE_PATH + "/game/websocket");
        TMController.conn.onmessage = e => TMController.receive(JSON.parse(e.data));
        TMController.conn.onerror = e => console.log(e); // TODO: close message? redirect to lobby?
        TMController.conn.onclose = e => console.log(e);

        // Init setup
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
        
        // Init lobby
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
            const game = msg.content.game;
            if (game.state == InLobbyState) {
                TMController.view(LobbyView);
                TMController.updateLobby(Object.assign({}, game.settings, {players: game.players}))
            } else {
                TMController.view(GameView);
            }
            break;
        case FirstGenCode:
            TMController.view(GameView);
            
        default:
            
        }
    },

    view(view) {
        const setupDiv = document.getElementById("setup");
        const lobbyDiv = document.getElementById("lobby");
        const gameDiv = document.getElementById("game");

        switch (view) {
        case SetupView:
            setupDiv.hidden = false;
            lobbyDiv.hidden = true;
            gameDiv.hidden = true;
            break;
        case LobbyView:
            setupDiv.hidden = true;
            lobbyDiv.hidden = false;
            gameDiv.hidden = true;
            break;
        case GameView:
            setupDiv.hidden = true;
            lobbyDiv.hidden = true;
            gameDiv.hidden = false;
            break;
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
    }

};

window.addEventListener('DOMContentLoaded', TMController.init);