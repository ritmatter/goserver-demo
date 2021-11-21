import React from 'react';

import CreateChannel from './create_channel.js';
import ChannelList from './channel_list.js';
import Message from './message.js';
import MessageList from './message_list.js';
import MessageSender from './message_sender.js';

class App extends React.Component {
  constructor(props) {
    super(props);

    this.state = {messages: [], channels: []};
    this.handleNewChannel = this.handleNewChannel.bind(this);

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

  fetchChannels() {
    fetch(window.location.href + "channels")
      .then(res => res.json())
      .then(
        (result) => {
          this.setState({
            channels: result
          });
        },
        (error) => {
          console.log(error);
        }
      )
  }

  componentDidMount() {
    this.fetchChannels();
  }

  handleNewChannel() {
    this.fetchChannels();
  }

  render() {
    return (
      <div className="App">
        <h1>YACA</h1>

        <div className="wrapper">
          <nav id="channelsBar">
            <div className="sidebar-header">
                <h3>Channels</h3>
                <ChannelList channels={this.state.channels} />
                <CreateChannel cb={this.handleNewChannel} />
            </div>
          </nav>

          <div id="messageContent">
            <MessageList messages={this.state.messages} />
            <MessageSender socket={this.socket} />
          </div>
        </div>
      </div>
    );
  }
}

export default App;
