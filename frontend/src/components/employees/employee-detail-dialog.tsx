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
      <DialogContent className="w-[calc(100%-2rem)] max-w-3xl">
        <DialogHeader>
          <DialogTitle>{employee?.name}</DialogTitle>
          <DialogDescription>
            Weekly working hours and time off — used by the solver to work out availability.
          </DialogDescription>
        </DialogHeader>
        {employee && (
          <DialogBody className="max-h-[70vh]">
            <Tabs defaultValue="schedule">
              {/* TabsList defaults to `sticky top-0` for long scrollable content,
                  but that collides with DialogBody's own top padding (py-4) —
                  the padding scrolls away first, so the grid's own header row
                  behind it briefly peeks out above the stuck tab bar. This
                  dialog's content is short enough not to need sticky tabs. */}
              <TabsList className="static">
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
          </DialogBody>
        )}
      </DialogContent>
    </Dialog>
  );
}
