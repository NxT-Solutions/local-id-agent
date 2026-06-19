import { useState } from "react";
import { LockKeyhole } from "lucide-react";
import { AlertBanner } from "@/components/layout/AlertBanner";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useAdminLock } from "@/context/AdminLockContext";

interface UnlockDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onUnlocked?: () => void;
}

export function UnlockDialog({ open, onOpenChange, onUnlocked }: UnlockDialogProps) {
  const { unlock } = useAdminLock();
  const [passcode, setPasscode] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(event: React.FormEvent) {
    event.preventDefault();
    setSubmitting(true);
    setError(null);

    try {
      await unlock(passcode);
      setPasscode("");
      onOpenChange(false);
      onUnlocked?.();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unlock failed");
    } finally {
      setSubmitting(false);
    }
  }

  function handleOpenChange(next: boolean) {
    if (!next) {
      setPasscode("");
      setError(null);
    }
    onOpenChange(next);
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <LockKeyhole className="h-4 w-4" />
            Unlock admin
          </DialogTitle>
          <DialogDescription>
            Enter the admin passcode to access settings, configuration, and
            integration tools.
          </DialogDescription>
        </DialogHeader>

        <form className="space-y-4" onSubmit={(event) => void handleSubmit(event)}>
          {error && (
            <AlertBanner variant="error" title="Unlock failed">
              {error}
            </AlertBanner>
          )}

          <div className="space-y-2">
            <Label htmlFor="admin-passcode">Admin passcode</Label>
            <Input
              id="admin-passcode"
              type="password"
              autoComplete="current-password"
              value={passcode}
              onChange={(event) => setPasscode(event.target.value)}
              autoFocus
            />
          </div>

          <div className="flex justify-end gap-2">
            <Button
              type="button"
              variant="outline"
              onClick={() => handleOpenChange(false)}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={submitting || passcode.length === 0}>
              {submitting ? "Unlocking…" : "Unlock"}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}
