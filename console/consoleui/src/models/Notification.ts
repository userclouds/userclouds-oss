export enum NotificationType {
  Info = 'info',
  Success = 'success',
  Alert = 'alert',
}

type Notification = {
  id: string; // TODO: uuid
  message: string;
  type: NotificationType;
};

export default Notification;
