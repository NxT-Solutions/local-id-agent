import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { Shield } from "lucide-react";
import { AlertBanner } from "@/components/layout/AlertBanner";
import { PageHeader } from "@/components/layout/PageHeader";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useAdminLock } from "@/context/AdminLockContext";

const MIN_PASSCODE_LENGTH = 8;

export function AdminSetupPage() {
  const { setup } = useAdminLock();
  const navigate = useNavigate();
  const [passcode, setPasscode] = useState("");
  const [confirmPasscode, setConfirmPasscode] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(event: React.FormEvent) {
    event.preventDefault();
    setError(null);

    if (passcode.length < MIN_PASSCODE_LENGTH) {
      setError(`Passcode must be at least ${MIN_PASSCODE_LENGTH} characters.`);
      return;
    }

    if (passcode !== confirmPasscode) {
      setError("Passcodes do not match.");
      return;
    }

    setSubmitting(true);
    try {
      await setup(passcode);
      navigate("/settings", { replace: true });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Setup failed");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="mx-auto flex min-h-screen max-w-lg flex-col justify-center px-4 py-8">
      <PageHeader
        title="Admin setup"
        description="Create an admin passcode to protect agent configuration on this device. The first user to complete this step becomes the admin for this installation."
      />

      <Card className="mt-6">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <Shield className="h-4 w-4" />
            Set admin passcode
          </CardTitle>
          <CardDescription>
            You will need this passcode to edit settings, run the auth demo, and
            view full diagnostics. End users can still monitor health and restart
            the agent without unlocking.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form className="space-y-4" onSubmit={(event) => void handleSubmit(event)}>
            {error && (
              <AlertBanner variant="error" title="Setup failed">
                {error}
              </AlertBanner>
            )}

            <div className="space-y-2">
              <Label htmlFor="setup-passcode">Admin passcode</Label>
              <Input
                id="setup-passcode"
                type="password"
                autoComplete="new-password"
                value={passcode}
                onChange={(event) => setPasscode(event.target.value)}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="setup-confirm">Confirm passcode</Label>
              <Input
                id="setup-confirm"
                type="password"
                autoComplete="new-password"
                value={confirmPasscode}
                onChange={(event) => setConfirmPasscode(event.target.value)}
              />
            </div>

            <Button
              type="submit"
              className="w-full"
              disabled={
                submitting ||
                passcode.length < MIN_PASSCODE_LENGTH ||
                confirmPasscode.length < MIN_PASSCODE_LENGTH
              }
            >
              {submitting ? "Saving…" : "Create admin passcode"}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
