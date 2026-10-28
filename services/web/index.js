const answer = "__ANSWER__";
const winSound = new Audio('/win.wav');
let attempts = 0;
const MAX = 6;
async function submit() {
    const input = document.getElementById('guess');
    const guess = input.value.toUpperCase();
    
    const data = await fetchResult(guess);
    console.log(data);

    if (guess.length !== 5) { setStatus('Must be 5 letters'); return; }
    const counts = {};
    for (const ch of answer) counts[ch] = (counts[ch] || 0) + 1;
    const states = new Array(5).fill('absent');
    for (let i = 0; i < 5; i++) {
        if (guess[i] === answer[i]) { states[i] = 'correct'; counts[guess[i]]--; }
    }
    for (let i = 0; i < 5; i++) {
        if (states[i] === 'absent' && counts[guess[i]] > 0) {
            states[i] = 'present'; counts[guess[i]]--;
        }
    }
    const row = document.createElement('div');
    row.className = 'row';
    for (let i = 0; i < 5; i++) {
        const cell = document.createElement('div');
        cell.className = 'cell ' + states[i];
        cell.textContent = guess[i];
        row.appendChild(cell);
    }
    document.getElementById('board').appendChild(row);
    input.value = '';
    attempts++;
    if (guess === answer) {
        winSound.currentTime = 0;
        winSound.play().catch(() => {});
    setStatus('You won in ' + attempts + '! Refresh for a new word.');
    input.disabled = true;
    } else if (attempts >= MAX) {
        setStatus('Out of attempts. The word was: ' + answer);
        input.disabled = true;
    }
}

function setStatus(m) { document.getElementById('status').textContent = m; }
document.getElementById('guess').addEventListener('keydown', e => {
    if (e.key === 'Enter') submit();
});

async function fetchResult(word) {
    const baseUrl = "/wordle"

    const params = new URLSearchParams({
        word: word
    });

    try {
        const response = await fetch(`${baseUrl}?${params}`);

        if (!response.ok) {
            throw new Error(`HTTP Error: Status: ${response.status}`);
        }

        return await response.json();;
    } catch (error) {
        console.log(`Error: `, error);
    }

    
}