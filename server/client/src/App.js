import React from 'react';

import Message from './message.js';
import MessageList from './message_list.js';
import MessageSender from './message_sender.js';

class App extends React.Component {
  constructor(props) {
    super(props);

    this.state = {messages: []};

    let socket = new WebSocket("ws://127.0.0.1:3000/ws");
    console.log("Attempting Connection...");

    socket.onopen = () => {
        console.log("Successfully Connected");
        socket.send("Hi From the Client!")
    };

    socket.onclose = event => {
        console.log("Socket Closed Connection: ", event);
        socket.send("Client Closed!")
    };

    socket.onerror = error => {
        console.log("Socket Error: ", error);
    };

    socket.onmessage = message => {
        console.log(message);
        let messages = this.state.messages;
        messages.push(<Message id={message.id} text={message.data}/>)
        this.setState({messages: messages});
    };

    this.socket = socket
  }

  render() {
    return (
      <div className="App">
        <h1>YACA</h1>
        <MessageList messages={this.state.messages} />
        <MessageSender socket={this.socket} />
      </div>
    );
  }
}

export default App;
