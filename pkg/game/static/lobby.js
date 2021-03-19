const JSON_DATA = JSON.parse(document.getElementById('json-data').innerHTML);
const BASE_PATH = JSON_DATA.base_path;
const BASE_URL = function() {
    let split = window.location.href.split("/")
    return `${split[0]}//${split[2]}`
}()

const AUTO_REFRESH_TIMEOUT = 30 * 1000;

let refreshTimer;

function writeLobbiesTable(lobbies) {
    const table = document.getElementById("lobbies-table");
    table.innerHTML = '<tr><th>Name</th><th>Current Players</th><th></th></tr>';
    for (let l of lobbies) {
      let row = document.createElement('tr');
      table.append(row);

      row.append(`<td>${l.lobby_name}</td>`);
      row.append(`<td>${Object.keys(l.players).length}</td>`);

      let buttons = document.createElement('td');
      row.append(buttons);

      let joinButton = document.createElement('button');
      buttons.append(joinButton);

      joinButton.innerText = "Join Game";
      joinButton.addEventListener('click', _ => window.location.href = `${BASE_URL}${BASE_PATH}/game?lobby=${l.lobby_name}`);
    }
}

function populateLobbies() {
    fetch(`${BASE_PATH}/lobby/list`).then((response) => {
        response.json().then((results) => {
            writeLobbiesTable(results);
        });
    });
}

function autoRefreshLobbies() {
    refreshTimer = setTimeout(_ => {
        populateLobbies();
        autoRefreshLobbies();
    }, AUTO_REFRESH_TIMEOUT);
}

function setup() {
    document.getElementById("new").addEventListener("click", _ => {
        window.location.href = `${BASE_URL}${BASE_PATH}/game?lobby=`;
    });

    document.getElementById("refresh").addEventListener("click", _ => {
        clearTimeout(refreshTimer);
        populateLobbies();
        autoRefreshLobbies();
    });

    populateLobbies();
    autoRefreshLobbies();

}

window.addEventListener('DOMContentLoaded', setup);