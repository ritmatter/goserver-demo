import React from 'react';

class Channel extends React.Component {
  render() {
    return (
      <div className="channel">
        <div>{this.props.name}</div>
      </div>
    );
  }
}

export default Channel;
