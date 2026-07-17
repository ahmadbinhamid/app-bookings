import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription,
  Tabs, TabsList, TabsTrigger, TabsContent,
} from "@flowposltd/ui";
import { DialogBody } from "@/components/ui/dialog-body";
import { type Employee } from "@/types";
import { EmployeeScheduleEditor } from "./employee-schedule-editor";
import { EmployeeTimeOffEditor } from "./employee-timeoff-editor";

interface EmployeeDetailDialogProps {
  open: boolean;
  onClose: () => void;
  locationId: string;
  employee: Employee | undefined;
}

export function EmployeeDetailDialog({ open, onClose, locationId, employee }: EmployeeDetailDialogProps) {
  return (
    <Dialog open={open} onOpenChange={(v) => !v && onClose()}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>{employee?.name}</DialogTitle>
          <DialogDescription>
            Weekly working hours and time off — used by the booking solver to work out availability.
          </DialogDescription>
        </DialogHeader>
        <DialogBody>
          {employee && (
            <Tabs defaultValue="schedule">
              <TabsList>
                <TabsTrigger value="schedule">Working hours</TabsTrigger>
                <TabsTrigger value="time-off">Time off</TabsTrigger>
              </TabsList>
              <TabsContent value="schedule" className="pt-4">
                <EmployeeScheduleEditor locationId={locationId} employeeId={employee.id} />
              </TabsContent>
              <TabsContent value="time-off" className="pt-4">
                <EmployeeTimeOffEditor locationId={locationId} employeeId={employee.id} />
              </TabsContent>
            </Tabs>
          )}
        </DialogBody>
      </DialogContent>
    </Dialog>
  );
}
