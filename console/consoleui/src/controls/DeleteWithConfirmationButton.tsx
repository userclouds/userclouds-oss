import {
  Dialog,
  DialogBody,
  DialogFooter,
  ButtonGroup,
  Button,
  IconDeleteBin,
  IconButton,
} from '@userclouds/ui-component-lib';
import { useLayoutEffect, useRef, useState } from 'react';

type DeleteConfirmationProps = {
  closeOnConfirm?: boolean;
  disabled?: boolean;
  icon?: React.ReactNode;
  id: string;
  message: string;
  onCancel?: () => void;
  onConfirmDelete: () => void;
  title: string;
};

const DeleteConfirmation = ({
  closeOnConfirm = true,
  disabled = false,
  icon = <IconDeleteBin />,
  id,
  message,
  onCancel,
  onConfirmDelete,
  title,
}: DeleteConfirmationProps) => {
  const dialogRef = useRef<HTMLDialogElement>(null);
  const [showDeleteConfirmation, setShowDeleteConfirmation] = useState(false);

  useLayoutEffect(() => {
    if (dialogRef.current?.open && !showDeleteConfirmation) {
      dialogRef.current?.close();
    } else if (!dialogRef.current?.open && showDeleteConfirmation) {
      dialogRef.current?.showModal();
    }
  }, [showDeleteConfirmation]);

  const handleCancel = () => {
    setShowDeleteConfirmation(false);
    onCancel?.();
  };

  const handleConfirmDelete = () => {
    closeOnConfirm && setShowDeleteConfirmation(false);
    onConfirmDelete();
  };

  return (
    <>
      <IconButton
        icon={icon}
        onClick={() => {
          setShowDeleteConfirmation(true);
        }}
        title="Delete"
        aria-label="Delete"
        disabled={disabled}
        size="small"
        id={id}
      />
      {showDeleteConfirmation && (
        <Dialog
          id="deleteConfirmationDialog"
          isDismissable={false}
          onClose={handleCancel}
          ref={dialogRef}
          title={title}
        >
          <>
            <DialogBody>{message}</DialogBody>
            <DialogFooter>
              <ButtonGroup>
                <Button
                  id="cancelDeleteButton"
                  onClick={handleCancel}
                  size="small"
                  theme="secondary"
                  type="button"
                >
                  Cancel
                </Button>
                <Button
                  id="confirmDeleteButton"
                  onClick={handleConfirmDelete}
                  size="small"
                  theme="primary"
                  type="button"
                >
                  Delete
                </Button>
              </ButtonGroup>
            </DialogFooter>
          </>
        </Dialog>
      )}
    </>
  );
};

export default DeleteConfirmation;
