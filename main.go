package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"slices"
	"sync"
)

func main() {
	_, has_cf_api_token := os.LookupEnv("CF_API_TOKEN")
	_, has_cf_identifier := os.LookupEnv("CF_IDENTIFIER")
	if !(has_cf_api_token && has_cf_identifier) {
		log.Fatalln("Please set CF_API_TOKEN and CF_IDENTIFIER")
		return
	}

	// Run both black_list and white_list in parallel
	var wg sync.WaitGroup
	black_list_set := make(map[string]bool)
	white_list_set := make(map[string]bool)
	wg.Add(2)
	go func() {
		defer wg.Done()
		black_list := read_domain_urls("lists.txt")
		black_list_set = convert_to_domain_set(black_list, true)
	}()
	go func() {
		defer wg.Done()
		white_list := read_domain_urls("whitelists.txt")
		white_list_set = convert_to_domain_set(white_list, true)
	}()
	wg.Wait()

	// black_list_set - white_list_set
	for k := range white_list_set {
		delete(black_list_set, k)
	}
	black_list_list := []string{}
	for k := range black_list_set {
		black_list_list = append(black_list_list, k)
	}

	// Sort alphabetically
	slices.Sort(black_list_list)

	log.Println("Total", len(black_list_list), "domains")

	// Write to file
	// file, err := os.Create("least_specific_domains.txt")
	// if err != nil {
	// 	fmt.Println(err.Error())
	// 	return
	// }
	// defer file.Close()
	// for _, v := range black_list_list {
	// 	file.WriteString(v + "\n")
	// }
	// return

	// Get cf lists
	prefix := "[AdBlock-DNS Block List]"
	cf_lists := get_cf_lists(prefix)

	// Compare existing policies
	sum := 0
	for _, v := range cf_lists {
		sum += int(v.(map[string]interface{})["count"].(float64))
	}
	if len(black_list_list) == sum {
		log.Println("Lists are the same size, skipping")
		return
	}

	policy_prefix := fmt.Sprintf("%s Block Ads", prefix)
	// deleted_policies = await cloudflare.delete_gateway_policy(policy_prefix)
	deleted_policy := delete_gateway_policy(policy_prefix)
	// fmt.Printf("Deleted %d gateway policies\n", deleted_policy)
	log.Println("Deleted", deleted_policy, "gateway policy")

	// Delete cf lists
	for _, v := range cf_lists {
		wg.Add(1)
		go func(name string, list_id string) {
			defer wg.Done()
			log.Println("Deleting list", name, "- ID:", list_id)
			delete_cf_list(list_id)
		}(v.(map[string]interface{})["name"].(string), v.(map[string]interface{})["id"].(string))
	}
	wg.Wait()

	// Create cf lists by 1000 chunks
	chunk_size := 1000
	chunk_counter := 0
	new_cf_lists_length := int(math.Ceil(float64(len(black_list_list)) / float64(chunk_size)))
	new_cf_lists := make([]interface{}, new_cf_lists_length)
	for i := 0; i < len(black_list_list); i += chunk_size {
		end := i + chunk_size
		if end > len(black_list_list) {
			end = len(black_list_list)
		}
		chunk_counter += 1
		name := fmt.Sprintf("%s %d", prefix, chunk_counter)
		wg.Add(1)
		go func(name string, list []string, cf_lists *[]interface{}, chunk_counter int) {
			defer wg.Done()
			log.Println("Creating list", name)
			cf_list := create_cf_list(name, list)
			(*cf_lists)[chunk_counter-1] = cf_list
		}(name, black_list_list[i:end], &new_cf_lists, chunk_counter)
	}
	wg.Wait()

	// Create cf policies
	cf_policies := get_gateway_policies(policy_prefix)
	new_cf_lists_ids := []string{}
	for _, v := range new_cf_lists {
		new_cf_lists_ids = append(new_cf_lists_ids, v.(map[string]interface{})["id"].(string))
	}
	if len(cf_policies) == 0 {
		log.Println("Creating firewall policy")
		create_gateway_policy(policy_prefix, new_cf_lists_ids)
	} else if len(cf_policies) != 1 {
		log.Println("More than one firewall policy found")
	} else {
		log.Println("Updating firewall policy")
		update_gateway_policy(policy_prefix, cf_policies[0].(map[string]interface{})["id"].(string), new_cf_lists_ids)
	}
	log.Println("Done!")
}
