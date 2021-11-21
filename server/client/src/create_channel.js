import React from 'react';

class CreateChannel extends React.Component {
  constructor(props) {
    super(props);
    this.createChannel = this.createChannel.bind(this);
  }

  createChannel() {
    let channel = {
      "name": document.getElementById("newChannel").value
    };

    fetch(window.location.href + 'channels/new', {
      method: 'post',
      body: JSON.stringify(channel)
    }).then(
        (result) => {
          document.getElementById("newChannel").value = "";
          this.props.cb();
        },
        (error) => {
          console.log(error);
        }
      )
  }

  render() {
    return (
      <div className="createChannel">
        <input type="text" id="newChannel" name="newChannel" />
        <input className="createButton" type="button" value="+" onClick={this.createChannel} />
      </div>
    );
  }
}

export default CreateChannel;
