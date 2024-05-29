function toMB(bytes) {
    let mb = bytes / 1024 / 1024
    mb = mb.toFixed(1)
    if (mb % 1.0 == 0) {
        return Math.floor(mb)
    }
    return mb
}

function addProgressBox(shareInfo) {
    let body = document.getElementsByClassName("cont").item(0)
    let bDown = toMB(shareInfo.bytesDown)
    let size = toMB(shareInfo.tSize) 
    let percent = Math.floor(shareInfo.bytesDown / shareInfo.tSize)
    
    body.innerHTML += `<div id="${shareInfo.itemId}" class="main-div">  
        <div class="left-con"> 
            <img src="icons/share.png">
        </div>
        <div class="right-con">
            <div class="first-line">
                <span class="file-name">${shareInfo.itemId}</span>

            </div>
            <div class="info">
                <span class="status">${shareInfo.status}</span>  
                <span class="peers-val">
                    <img class="icon" src="icons/community.png" alt="an image">
                    <span class="peers">${shareInfo.peerNum}</span>
                </span>
                <span class="file-size">
                    <img class="icon" src="icons/download.png" alt="An image">
                    <span class="dinfo">${bDown}  MB/${size}  MB</span>
                </span>
                <span class="percent">${percent}%</span>
            </div>
            <div class="progress-box">
                <div class="progress" style="width: ${percent}%;"></div>
            </div>
        </div>
    </div>`
}

function createErrorBox(message) {
    let errorLine = document.getElementById("error-line")
    errorLine.innerHTML = `<div class="error">
                <span class="text">${message}</span>
                <div class="line"></div>
            </div>`
}

function updateBox(info) {
    let bDown = toMB(info.bytesDown)
    let size = toMB(info.tSize) 
    let transferBox = document.getElementById(info.itemId)
    if (transferBox == null || transferBox == undefined) {
        addProgressBox(info)
        return true
    }
    let peers = transferBox.getElementsByClassName("peers").item(0)
    peers.textContent = info.peerNum
    // if (info.status == "Sharing") {
    //     return
    // }
    let status = transferBox.getElementsByClassName("status").item(0)
    let dinfo = transferBox.getElementsByClassName("dinfo").item(0)
    let percentbox = transferBox.getElementsByClassName("percent").item(0)
    let pbar = transferBox.getElementsByClassName("progress").item(0)
    let percent = Math.floor(info.bytesDown * 100 / info.tSize)
    status.textContent = info.status
    dinfo.textContent = `${bDown}  MB/${size}  MB`
    percentbox.textContent = `${percent}%`
    pbar.style.width = `${percent}%`
}

function update(jsonStr) {
    let arry = JSON.parse(jsonStr)
    if (arry) {
        arry.forEach(info => {
            updateBox(info)
        });
    }
}