// Imported Constants
const JSON_DATA = JSON.parse(document.getElementById('json-data').innerHTML)
const BASE_PATH = JSON_DATA.base_path
const LOBBY = window.location.href.split("?lobby=")[1];
const IS_HOST = LOBBY === "";


// Message Code Constants
// Two way codes
const PingCode = "ping";
const PongCode = "pong";

// Sending Codes
const HostGameCode = "host_game";
const JoinGameCode = "join_game";
const UpdateLobbySettingsCode = "update_lobby_settings";
const StartGameCode = "start_game";
const PassedCardsCode = "passed_cards";
const PlayedCardCode = "played_card";

// Recieving Codes
const InfoCode = "info";
const UpdateLobbyCode = "update_lobby";
const UpdateCode = "update";
const PassCardsCode = "pass_cards";
const PlayCardCode = "play_card";


// State Constants
const SetupState = "setup";
const InLobbyState = "in_lobby";
const InGameState = "in_game";


// Images
let clubsImage = document.getElementById("clubs-img");
let heartsImage = document.getElementById("hearts-img");
let spadesImage = document.getElementById("spades-img");
let diamondsImage = document.getElementById("diamonds-img");


function createCard(card) {
    let c = document.createElement("div");
    c.classList.add("card");

    suit = document.createElement("div");
    c.append(suit);
    suit.classList.add("suit");
    switch (card.suit) {
    case "clubs":
        suit.append(clubsImage.cloneNode(true));
        break;

    case "hearts":
        suit.append(heartsImage.cloneNode(true));
        break;

    case "spades":
        suit.append(spadesImage.cloneNode(true));
        break;

    case "diamonds":
        suit.append(diamondsImage.cloneNode(true));
        break;
    }
    
    value = document.createElement("div");
    c.append(value);
    value.classList.add("value");
    switch (card.value) {
        case "ace":
            value.innerText = "A";
            break;
    
        case "king":
            value.innerText = "K";
            break;
    
        case "queen":
            value.innerText = "Q";
            break;
    
        case "jack":
            value.innerText = "J";
            break;

        default:
            value.innerText = card.value;
        }
    

    return c;
}

function updatePlayerDashboard(container, name, playerInfo) {
    container.innerHTML = "";
    container.classList.toggle("leader", playerInfo.lead);

    const nameDiv = document.createElement("div");
    container.append(nameDiv);
    nameDiv.innerText = name
    if (playerInfo.lead) {
        nameDiv.innerText = `Lead: ${nameDiv.innerText}`;
    }
    

    const score = document.createElement("div");
    container.append(score);
    score.classList.add("score");
    score.innerHTML = `Score: ${playerInfo.score}`;

    const roundPoints = document.createElement("div");
    container.append(roundPoints);
    roundPoints.classList.add("hidden-cards");
    roundPoints.innerHTML = `Round Points: ${playerInfo.roundPoints}`;

    const cards = document.createElement("div");
    container.append(cards);
    cards.classList.add("hidden-cards");
    cards.innerHTML = `Cards: ${playerInfo.numCards}`;
}

// Controller
let CardsController = {

    init() {
        // Init connection
        const urlSplit = window.location.href.split("/");
        const protocol = urlSplit[0] == "http:" ? "ws" : "wss";
        const hostAndPort = urlSplit[2];

        CardsController.conn = new WebSocket(protocol + "://" + hostAndPort + BASE_PATH + "/game/websocket");
        CardsController.conn.onmessage = e => {
            CardsController.receive(JSON.parse(e.data))
        };
        CardsController.conn.onerror = e => console.log(e); // TODO: close message? redirect to lobby?
        CardsController.conn.onclose = e => console.log(e);


        CardsController.view(SetupState);
    },

    send(msg) {
        CardsController.conn.send(JSON.stringify(msg))
    },

    receive(msg) {
        if (msg.code != PingCode) console.log(msg);
        switch(msg.code) {
        case PingCode:
            CardsController.send({id: msg.id, code: PongCode});
            break;

        case InfoCode:
            if (CardsController.currentState == InGameState) {
                document.getElementById("info-message").innerText = `Info: ${msg.content}`;
            }
            break;

        case UpdateLobbyCode:
            CardsController.view(InLobbyState);
            CardsController.updateLobby(msg.content.settings.max_points, msg.content.players);
            break;

        case UpdateCode:
            CardsController.view(InGameState);
            CardsController.updateGame(msg.content);
            break;

        case PassCardsCode:
            CardsController.passCards();
            break;

        case PlayCardCode:
            CardsController.playCard();
            break;
            
        default:
            
        }
    },

    view(state) {
        if (CardsController.currentState == state) {
            return
        }
        CardsController.currentState = state;


        document.querySelectorAll("main > div").forEach(d => {
            if (!d.classList.contains("hidden")) d.classList.add("hidden");
        });

        switch (state) {
        case SetupState:
            document.getElementById("setup-view").classList.remove("hidden");
            CardsController.initSetup();
            break;
        case InLobbyState:
            document.getElementById("lobby-view").classList.remove("hidden");
            CardsController.initLobby();
            break;
        case InGameState:
            document.getElementById("game-view").classList.remove("hidden");
            CardsController.initGame();
            break;
        }
    },

    //-----------------------------------------------------------
    //------------------------  Setup   -------------------------
    //----------------------------------------------------------

    initSetup() {
        if (CardsController.doneSetup) {
            return;
        }
        CardsController.doneSetup = true;

        if (IS_HOST) {
            document.getElementById("submit-setup").onclick = function() {
                const lobbyName = document.getElementById("lobby-name-input").value;
                const nickname = document.getElementById("nickname-input").value;
                CardsController.nickname = nickname;
                CardsController.send({code: HostGameCode, content: {
                    lobby_name: lobbyName,
                    nickname: nickname
                }});
            };
        } else {
            document.getElementById("lobbyname").hidden = true;
            document.getElementById("lobby-name-input").required = false;

            document.getElementById("submit-setup").onclick = function() {
                const nickname = document.getElementById("nickname-input").value;
                CardsController.nickname = nickname;
                CardsController.send({code: JoinGameCode, content: {
                    lobby: LOBBY,
                    nickname: nickname
                }});
                
            };
        }
    },

    //-----------------------------------------------------------
    //------------------------  Lobby   -------------------------
    //-----------------------------------------------------------

    initLobby() {
        if (CardsController.doneLobby) {
            return;
        }
        CardsController.doneLobby = true;

        if (IS_HOST) {
            const maxPointsInput = document.getElementById("max-points-input");
            maxPointsInput.disabled = false;
            maxPointsInput.addEventListener("change", _ => {
                CardsController.send({code: UpdateLobbySettingsCode, content: {
                    max_points: parseInt(maxPointsInput.value)
                }});
            });

            const cpuNameLabel = document.getElementById("cpu-name-label");
            cpuNameLabel.hidden = false;
            const cpuNameInput = document.getElementById("cpu-name-input");
            cpuNameInput.hidden = false;
            const addCPUButton = document.getElementById("add-cpu");
            addCPUButton.hidden = false;
            addCPUButton.addEventListener("click", _ => {
                CardsController.send({code: UpdateLobbySettingsCode, content: {
                    add_cpu: cpuNameInput.value
                }});
            });

            const startGameButton = document.getElementById("start-game");
            startGameButton.hidden = false;
            startGameButton.addEventListener("click", _ => {
                CardsController.send({code: StartGameCode});
            });
        }
    },

    updateLobby(maxPoints, players) {
        const maxPointsInput = document.getElementById("max-points-input");
        maxPointsInput.value = maxPoints;

        
        const playerList = document.getElementById("player-list");
        playerList.innerHTML = "";
        let items = [];
        let buttons = [];
        for (let p of players) {
            let pItem = document.createElement("li");
            playerList.append(pItem);
            items.push(pItem);
            if (p.cpu) {
                pItem.innerHTML += `<div class="cpu">CPU</div>`
            }
            pItem.innerHTML += `<span>${p.name}</span>`;
            if (IS_HOST) {
                let swapButton = document.createElement("button");
                pItem.append(swapButton);
                swapButton.innerText = "Swap";
                buttons.push(swapButton);

                if (p.cpu) {
                    let removeButton = document.createElement("button");
                    pItem.append(removeButton);
                    removeButton.innerText = "Remove";
                    removeButton.addEventListener("click", function() {
                        CardsController.send({code: UpdateLobbySettingsCode, content: {
                            remove_cpu: p.name
                        }})
                    })
                }
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
                            CardsController.send({code: UpdateLobbySettingsCode, content: {
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
    //-----------------------  In Game   ------------------------
    //-----------------------------------------------------------

    initGame() {
        if (CardsController.doneGame) {
            return;
        }
        CardsController.doneGame = true;
    },

    updateGame(data) {
        let playerIndex = data.playerOrder.indexOf(CardsController.nickname);
        let leftPlayer = data.playerOrder[(playerIndex + 1) % 4];
        let acrossPlayer = data.playerOrder[(playerIndex + 2) % 4];
        let rightPlayer = data.playerOrder[(playerIndex + 3) % 4];

        leftPlayerDiv = document.getElementById("player-left");
        updatePlayerDashboard(leftPlayerDiv, leftPlayer, data.playerInfo[leftPlayer]);

        const acrossPlayerDiv = document.getElementById("player-across");
        updatePlayerDashboard(acrossPlayerDiv, acrossPlayer, data.playerInfo[acrossPlayer]);

        const rightPlayerDiv = document.getElementById("player-right");
        updatePlayerDashboard(rightPlayerDiv, rightPlayer, data.playerInfo[rightPlayer]);

        const currentTrickDiv = document.getElementById("current-trick");
        currentTrickDiv.innerHTML = "";
        if (data.currentTrick) {
            for (const c of data.currentTrick) {
                currentTrickDiv.append(createCard(c));
            }
        }

        const playerInfoDiv = document.getElementById("player-info");
        playerInfoDiv.innerHTML = "";
        const playerInfoContainer = document.createElement("div");
        playerInfoDiv.append(playerInfoContainer);
        playerInfoContainer.classList.add("info-bar")
        console.log(data)
        console.log(playerIndex)
        updatePlayerDashboard(playerInfoContainer, CardsController.nickname, data.playerInfo[CardsController.nickname]);
        
        const playerHandDiv = document.createElement("div");
        playerInfoDiv.append(playerHandDiv);
        playerHandDiv.classList.add("hand");
        for (let i = 0; i < data.hand.length; i++) {
            const c = data.hand[i];
            const playerCard = document.createElement("div");
            playerCard.classList.add("playerCard");
            playerCard.append(createCard(c));
            const pickCardButton = document.createElement("button");
            pickCardButton.innerText = "Pick";
            playerCard.append(pickCardButton);
            playerCard.index = i;
            pickCardButton.addEventListener("click", function() {
                if (CardsController.passingCards) {
                    playerCard.classList.toggle("passedCard");
                }
                if (CardsController.playingCard) {
                    CardsController.playingCard = false;
                    CardsController.send({code: PlayedCardCode, content: {
                        card: playerCard.index
                    }});
                }
            })

            playerHandDiv.append(playerCard);
        }
        const cards = document.querySelectorAll(".playerCard");
        const passButton = document.createElement("button");
        passButton.innerText = "Pass Cards";
        passButton.id = "pass-button";
        passButton.classList.add("hidden");
        passButton.addEventListener("click", function() {
            let selected = [];
            for (const c of cards) {
                if (c.classList.contains("passedCard")) {
                    selected.push(c.index);
                }
            } 
            if (selected.length == 3) {
                CardsController.passingCards = false;
                document.getElementById("pass-button").classList.add("hidden");
                CardsController.send({code: PassedCardsCode, content: {
                    cards: selected
                }});
            }
        });
        playerHandDiv.append(passButton);

        document.getElementById("max-points-info").innerText = `Max Points: ${data.maxPoints}`;
        document.getElementById("pass-direction-info").innerText = `Pass Direction: ${data.passDirection}`;
    },

    passCards() {
        CardsController.passingCards = true;
        document.getElementById("pass-button").classList.remove("hidden");
    },

    playCard() {
        CardsController.playingCard = true;
    }
};

CardsController.init();