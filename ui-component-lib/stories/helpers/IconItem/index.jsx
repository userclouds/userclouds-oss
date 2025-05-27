import React from 'react';
import { CopyToClipboard } from 'react-copy-to-clipboard';

function IconItem({ icon, name }) {
  const componentText = `<${icon.props.mdxType} />`;

  return (
    <CopyToClipboard text={componentText}>
      <div className="icon-item">
        {icon}
        <div className="icon-name">{name}</div>
      </div>
    </CopyToClipboard>
  );
}

export default IconItem;
