import Notification from '../models/Notification';

export const POST_NEW_NOTIFICATION = 'POST_NEW_NOTIFICATION';
export const REMOVE_POSTED_NOTIFICATION = 'REMOVE_POSTED_NOTIFICATION';

export const postNewNotification = (notification: Notification) => ({
  type: POST_NEW_NOTIFICATION,
  data: notification,
});

export const removePostedNotification = (id: string) => ({
  type: REMOVE_POSTED_NOTIFICATION,
  data: id,
});
