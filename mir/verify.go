package mir

import "fmt"

// Verify checks value references, duplicate results and region outputs.
func Verify(module *Module) error {
	if module == nil || module.Entry == nil {
		return fmt.Errorf("MIR module has no entry region")
	}
	defined := map[ValueID]bool{}
	for _, decl := range module.Declarations {
		if err := verifyInstruction(decl, defined); err != nil {
			return err
		}
	}
	for _, fn := range module.Functions {
		if fn == nil || fn.Body == nil {
			return fmt.Errorf("MIR function has no body")
		}
		if err := verifyRegion(fn.Body, cloneDefined(defined)); err != nil {
			return fmt.Errorf("function %s: %w", fn.Name, err)
		}
	}
	return verifyRegion(module.Entry, defined)
}

func verifyRegion(region *Region, inherited map[ValueID]bool) error {
	if region == nil {
		return fmt.Errorf("nil MIR region")
	}
	defined := cloneDefined(inherited)
	for _, inst := range region.Instructions {
		for _, arg := range inst.Args {
			if arg != 0 && !defined[arg] {
				return fmt.Errorf("instruction %d uses undefined value %%%d", inst.ID, arg)
			}
		}
		for _, nested := range inst.Regions {
			if err := verifyRegion(nested, defined); err != nil {
				return fmt.Errorf("instruction %d: %w", inst.ID, err)
			}
		}
		if inst.Result != 0 {
			if defined[inst.Result] {
				return fmt.Errorf("value %%%d is defined more than once", inst.Result)
			}
			defined[inst.Result] = true
		}
	}
	if region.Result != 0 && !defined[region.Result] {
		return fmt.Errorf("region %s returns undefined value %%%d", region.Name, region.Result)
	}
	return nil
}

func verifyInstruction(inst *Instruction, defined map[ValueID]bool) error {
	if inst == nil {
		return fmt.Errorf("nil MIR instruction")
	}
	for _, arg := range inst.Args {
		if arg != 0 && !defined[arg] {
			return fmt.Errorf("declaration %d uses undefined value %%%d", inst.ID, arg)
		}
	}
	if inst.Result != 0 {
		if defined[inst.Result] {
			return fmt.Errorf("value %%%d is defined more than once", inst.Result)
		}
		defined[inst.Result] = true
	}
	return nil
}

func cloneDefined(source map[ValueID]bool) map[ValueID]bool {
	result := make(map[ValueID]bool, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
