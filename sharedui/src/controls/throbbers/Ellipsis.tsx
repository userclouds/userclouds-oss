import Styles from './Throbbers.module.css';

const Ellipsis = ({ id }: { id: string }) => {
  return (
    <div id={id} className={Styles.ellipsis}>
      <div />
      <div />
      <div />
      <div />
    </div>
  );
};

export default Ellipsis;
