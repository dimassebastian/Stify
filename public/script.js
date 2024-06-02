// script.js
export function handleAfterRequest(event) {
    // Isi dengan logika Anda
  
document.body.addEventListener('htmx:afterRequest', function(event) {
    if (event.detail.target.id === 'tracks') {
        const tracksDiv = event.detail.target;

        // Clear previous content
        tracksDiv.innerHTML = '';

        // Ensure the response is parsed correctly
        let topTracks;
        try {
            topTracks = JSON.parse(event.detail.xhr.responseText);
            console.log('Parsed response:', topTracks);
        } catch (e) {
            console.error('Failed to parse JSON:', e);
            return;
        }

        // Create and append cards for each track
        topTracks.forEach(function(track) {
            const card = document.createElement('div');
            card.classList.add('card');

            const img = document.createElement('img');
            img.src = track.img;
            img.alt = track.title + ' - ' + track.artist;

            const title = document.createElement('h3');
            title.textContent = track.title;

            const artist = document.createElement('p');
            artist.textContent = track.artist;

            card.appendChild(img);
            card.appendChild(title);
            card.appendChild(artist);

            // Debugging: Log each card element to the console
            console.log('Created card:', card);

            tracksDiv.appendChild(card);
        });
    }
});

}
