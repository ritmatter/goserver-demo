import React from 'react';

class MessageSender extends React.Component {
  constructor(props) {
    super(props);
    this.sendMessage = this.sendMessage.bind(this);
  }

  sendMessage() {
    let message = document.getElementById("messageCompose").value;
    try {
      this.props.socket.send(message);
    }
    catch(err) {
      console.log(err);
    }
    return true;
  }

  render() {
    return (
      <div className="messageSender">
        <input type="text" id="messageCompose" name="messageCompose" />
        <input className="sendButton" type="button" value="Send" onClick={this.sendMessage} />
      </div>
    );
  }
}

export default MessageSender;
