import React from 'react';

function ColorPalette({ colorCategory, colorData, singleHexColor }) {
  const colorNumbers = [
    '50',
    '100',
    '200',
    '300',
    '400',
    '500',
    '600',
    '700',
    '800',
    '900',
  ];

  const match = (hexValue, number) => (
    <>
      <div className="swatch-value">{number}</div>
      <div className="swatch-example" style={{ background: hexValue }} />
      <div className="swatch-hex">{hexValue}</div>
      <div className="swatch-var">
        $color-{colorCategory}-{number}
      </div>
    </>
  );

  const nomatch = (number) => (
    <>
      <div className="swatch-value">{number}</div>
      <div className="swatch-example" />
    </>
  );

  // Iterate over "colorNumbers" and see if there are matches in the data
  // Extract the matching data if a match, otherwise show empty swatch

  return (
    <div className="swatch-wrap">
      {singleHexColor ? (
        <div className="swatch">
          <div
            className="swatch-example"
            style={{ background: singleHexColor }}
          />
          <div className="swatch-hex">{singleHexColor}</div>
          <div className="swatch-var">$color-{colorCategory}</div>
        </div>
      ) : (
        <>
          {colorNumbers.map((colorNumber) => (
            <div className="swatch" key={colorNumber}>
              {Object.keys(colorData).includes(colorNumber) ? (
                <>{match(colorData[colorNumber], colorNumber)}</>
              ) : (
                <>{nomatch(colorNumber)}</>
              )}
            </div>
          ))}
        </>
      )}
    </div>
  );
}

export default ColorPalette;
