const CORRECT = "correct";
const PRESENT = "present";
const ABSENT = "absent";

const winSound = new Audio('/win.wav');
let attempts = 0;
const MAX = 6;
async function submit() {
    const input = document.getElementById('guess');
    const guess = input.value.toUpperCase();
    if (guess.length !== 5) { setStatus('Must be 5 letters'); return; }

    const data = await fetchResult(guess);
    const states = data.Results;

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
    if (checkCorrect(states)) {
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

function checkCorrect(states) {
    for (const s of states) {
        if (s != CORRECT) return false;
    }
    return true;
}