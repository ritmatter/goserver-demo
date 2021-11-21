import React from 'react';

class MessageList extends React.Component {
  render() {
    const messages = [];
    for (const message of this.props.messages) {
      messages.push(<li key={message.props.id}>{message}</li>);
    }
    return (
      <div className="messages-list">
        <ul>
          {messages}
        </ul>
      </div>
    );
  }
}

export default MessageList;
