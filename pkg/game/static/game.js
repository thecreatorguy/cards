const JSON_DATA = JSON.parse(document.getElementById('json-data').innerHTML)
const BASE_PATH = JSON_DATA.base_path

const SetupView = "state";
const LobbyView = "lobby";
const GameView = "game";

const HostGameCode = "host_game";
const JoinGameCode = "join_game";
const SyncGameCode = "sync_game";

const LOBBY = window.location.href.split("?lobby=")[1];
const IS_HOST = LOBBY === "";

let TMController = {

    init() {
        // Init connection
        const urlSplit = window.location.href.split("/");
        const protocol = urlSplit[0] == "http:" ? "ws" : "wss";
        const hostAndPort = urlSplit[2];

        TMController.conn = new WebSocket(protocol + "://" + hostAndPort + BASE_PATH + "/game/websocket");
        TMController.conn.onopen = _ => {
            if (IS_HOST) {
                TMController.view(SetupView)
            } else {
                TMController.conn.send(JSON.stringify({code: SyncGameCode}));
            }
        };
        TMController.conn.onmessage = e => TMController.recieve(JSON.parse(e.data));
        TMController.conn.onerror = e => console.log(e); // TODO: close message? redirect to lobby?

        // Init setup
        if (!IS_HOST) {
            const lobbyName = document.getElementById("lobbyname");
            lobbyName.hidden = true;

            const lobbyNameInput = document.getElementById("lobby-name-input");
            lobbyNameInput.required = false;
        }
        document.getElementById("submit-setup").onclick = function() {
            if (IS_HOST) {
                const lobbyName = document.getElementById("lobby-name-input").value;
                const nickname = document.getElementById("nickname-input").value;
                TMController.conn.send(JSON.stringify({code: HostGameCode, content: {
                    lobby_name: lobbyName,
                    nickname: nickname
                }}));
            } else {
                const nickname = document.getElementById("nickname-input").value;
                TMController.conn.send(JSON.stringify({code: JoinGameCode, content: {
                    lobby_name: LOBBY,
                    nickname: nickname
                }}));
            }
        };

        // Init lobby
    },

    receive(msg) {
        switch(msg.code) {
        default:
            console.log(message);
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
    }

};

window.addEventListener('DOMContentLoaded', TMController.init);