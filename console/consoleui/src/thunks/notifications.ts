import { v4 as uuidv4 } from 'uuid';
import { AppDispatch } from '../store';
import Notification, { NotificationType } from '../models/Notification';
import {
  postNewNotification,
  removePostedNotification,
} from '../actions/notifications';

const DEFAULT_NOTIFICATION_DURATION = 5000;

const postToast =
  (message: string, type: NotificationType, duration?: number) =>
  (dispatch: AppDispatch) => {
    const notification: Notification = {
      id: uuidv4(),
      type,
      message,
    };
    dispatch(postNewNotification(notification));

    setTimeout(() => {
      dispatch(removePostedNotification(notification.id));
    }, duration || DEFAULT_NOTIFICATION_DURATION);
  };

export const postInfoToast =
  (message: string, duration?: number) => (dispatch: AppDispatch) => {
    dispatch(postToast(message, NotificationType.Info, duration));
  };

export const postSuccessToast =
  (message: string, duration?: number) => (dispatch: AppDispatch) => {
    dispatch(postToast(message, NotificationType.Success, duration));
  };

export const postAlertToast =
  (message: string, duration?: number) => (dispatch: AppDispatch) => {
    dispatch(postToast(message, NotificationType.Alert, duration));
  };
