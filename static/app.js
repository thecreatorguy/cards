const JSON_DATA = JSON.parse(document.getElementById('json-data').innerHTML)
const BASE_URL = JSON_DATA.base_url

class TerraformingMars {
    constructor() {
        
    }
}

function handle

function main() {
    const urlSplit = window.location.href.split("/");
    const protocol = urlSplit[0] == "http:" ? "ws" : "wss";
    const hostAndPort = urlSplit[2]

    const ws = new WebSocket(protocol + "://" + hostAndPort + BASE_URL + "/game");
    ws.onopen = _ => {
        ws.send("hello");
    };
    ws.onmessage = e => console.log(e.data);
}

window.addEventListener('DOMContentLoaded', _ => main());