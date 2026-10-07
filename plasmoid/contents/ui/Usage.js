function resetText(timestamp, now) {
    if (!timestamp || timestamp <= 0)
        return "Reset time unavailable";
    var minutes = Math.ceil((timestamp * 1000 - now) / 60000);
    if (minutes <= 0)
        return "Reset pending refresh";
    if (minutes >= 1440) {
        var days = Math.floor(minutes / 1440);
        return "Resets in " + days + (days === 1 ? " day" : " days");
    }
    var hours = Math.floor(minutes / 60);
    return "Resets in " + (hours > 0 ? hours + "h " : "") + (minutes % 60) + "m";
}

function updatedText(timestamp, now) {
    if (!timestamp)
        return "Last updated: never";
    var seconds = Math.max(0, Math.floor((now - timestamp) / 1000));
    if (seconds === 0)
        return "Last updated: just now";
    if (seconds < 60)
        return "Last updated: " + seconds + " sec ago";
    if (seconds < 3600)
        return "Last updated: " + Math.floor(seconds / 60) + " min ago";
    if (seconds < 86400)
        return "Last updated: " + Math.floor(seconds / 3600) + "h ago";
    return "Last updated: " + Math.floor(seconds / 86400) + " days ago";
}
