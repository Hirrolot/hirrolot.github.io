window.addEventListener("load", function (event) {
    genAuthorEmoji();
});

function genAuthorEmoji() {
    const emojis = ["💋", "💄", "💅", "❤️", "👄", "👅", "👠", "👗", "🌹", "🌺", "🪷", "🌷", "💐", "🫒", "🥀", "🌼", "🌱", "🌳", "🌴", "🍁", "🍄", "🌹", "🥩", "🍎", "🍒", "🍓", "🍇", "🍉", "🍌", "🍫", "🍰", "🍭", "🍬", "🍧", "🍝", "🎈", "💕", "💖", "💘", "💞", "💓", "💝", "💗", "❤️‍🔥", "🫀", "🎀", "🎁", "🌈", "🦄", "🔮", "🎑", "🧸", "🐬", "🦔", "🦇", "🦨", "🦬", "🐌", "🐞", "🦋", "🌀", "🎎", "🪆", "🚩", "🛕", "⛪", "🏰", "🐚", "👁️", "🎼", "🪽🪽🪽", "🕊️", "🗽", "🪐", "🎅", "🎄", "☃️", "🌌", "🏜️", "🌉", "🌃", "🌆", "🌇", "🏙", "🎆", "🎑", "🪴", "⛲", "🪅", "🎨", "🍹", "⛩️", "🎭", "🥘"];

    const randomEmoji = document.createElement("span");
    randomEmoji.textContent = emojis[Math.floor(Math.random() * emojis.length)];
    randomEmoji.innerHTML += "&nbsp;";

    const blogLink = document.createElement("a");
    blogLink.href = "..";
    blogLink.textContent = "hirrolot's Blog";

    const fancyAuthor = document.createElement("p");
    fancyAuthor.className = "author";
    fancyAuthor.appendChild(randomEmoji);
    fancyAuthor.appendChild(blogLink);

    const author = document.querySelector(".author");

    if (author) {
        author.replaceWith(fancyAuthor);
    }
}
